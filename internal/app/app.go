package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/command"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/command/commandimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram/api_adapter"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	paserimpl "github.com/orgball2608/insta-parser-telegram-bot/internal/parser/parserimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/ratelimit"
	repositories "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/fx"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram/telegramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/pgx"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

var Module = fx.Options(
	fx.Provide(
		config.New,
		logger.FxOption,
		pgx.New,
		newHTTPServer,
		api_adapter.NewPlaywrightManager,
		// Rate limiter provider
		func() ratelimit.Limiter {
			// Allow 1 heavy command every 10 seconds, with a burst of 2 commands
			return ratelimit.NewInMemoryLimiter(1, 10*time.Second, 2)
		},
	),
	fx.Provide(
		fx.Annotate(
			telegramimpl.New,
			fx.As(new(telegram.Client)),
		),
		fx.Annotate(
			api_adapter.New,
			fx.As(new(instagram.Client)),
		),
		fx.Annotate(
			paserimpl.New,
			fx.As(new(parser.Client)),
		),
		fx.Annotate(
			commandimpl.New,
			fx.As(new(command.Client)),
		),
	),
	repositories.Module,
	fx.Invoke(runMigrations),
	fx.Invoke(registerHTTPRoutes),
	fx.Invoke(startServices),
)

type HTTPServer struct {
	server *http.Server
	log    logger.Logger
}

func newHTTPServer(log logger.Logger, cfg *config.Config) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.App.Port),
			ReadTimeout:  cfg.App.Timeout,
			WriteTimeout: cfg.App.Timeout,
		},
		log: log,
	}
}

func registerHTTPRoutes(server *HTTPServer) {
	router := http.NewServeMux()
	router.HandleFunc("GET /healthz", server.healthCheckHandler)
	server.server.Handler = router
}

func (s *HTTPServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	s.log.Info("Health check request received", "method", r.Method, "url", r.URL.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func runMigrations(log logger.Logger, cfg *config.Config) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database connection", "error", err)
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := goose.Up(db, filepath.Join(wd, "migrations")); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func startServices(
	lc fx.Lifecycle,
	log logger.Logger,
	server *HTTPServer,
	cmdClient command.Client,
	pClient parser.Client,
) {
	g, gCtx := errgroup.WithContext(context.Background())

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-sigChan:
			log.Info("Received shutdown signal from OS", "signal", sig)
		case <-gCtx.Done():
		}
	}()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting services...")

			g.Go(func() error {
				log.Info("Starting HTTP server", "addr", server.server.Addr)
				if err := server.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					return fmt.Errorf("http server failed: %w", err)
				}
				return nil
			})

			g.Go(func() error {
				return cmdClient.HandleCommand(gCtx)
			})

			g.Go(func() error {
				return pClient.ScheduleParseStories(gCtx)
			})

			g.Go(func() error {
				log.Info("Starting database cleanup scheduler")
				return pClient.ScheduleDatabaseCleanup(gCtx)
			})

			g.Go(func() error {
				log.Info("Starting post checking scheduler")
				return pClient.SchedulePostChecking(gCtx)
			})

			// Goroutine to wait for the first service to fail and initiate shutdown
			go func() {
				if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
					log.Error("A critical service failed, application is shutting down", "error", err)
				} else {
					log.Info("All services have been shut down.")
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Initiating graceful shutdown...")
			return server.server.Shutdown(ctx)
		},
	})
}
