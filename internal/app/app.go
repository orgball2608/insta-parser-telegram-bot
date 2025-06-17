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
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser/paserimpl"
	repositories "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/fx"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram/telegramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/pgx"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		config.New,
		logger.FxOption,
		pgx.New,
		newHTTPServer,
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
	tgClient telegram.Client,
	pClient parser.Client,
	cmdClient command.Client,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Info("Starting HTTP server", "addr", server.server.Addr)
				if err := server.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("HTTP server failed", "error", err)
				}
			}()

			go func() {
				if err := pClient.ScheduleParseStories(ctx); err != nil {
					log.Error("Story parser failed", "error", err)
					tgClient.SendMessageToUser(fmt.Sprintf("Story parser error: %v", err))
				}
			}()

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if err := cmdClient.HandleCommand(); err != nil {
							log.Error("Command handler failed", "error", err)
							tgClient.SendMessageToUser(fmt.Sprintf("Command error: %v", err))
						}
					}
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Shutting down HTTP server")
			return server.server.Shutdown(ctx)
		},
	})

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		log.Info("Received shutdown signal", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.server.Shutdown(ctx); err != nil {
			log.Error("Server shutdown failed", "error", err)
		}
	}()
}
