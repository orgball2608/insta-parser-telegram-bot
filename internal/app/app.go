package app

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/command"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/command/commandimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram/instagramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser/paserimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/pgx"
	repositories "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/fx"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram/telegramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
	"net/http"
	"os"
	"path/filepath"
)

var App = fx.Options(
	fx.Provide(
		config.New,
		logger.FxOption,
		pgx.New,
	),
	fx.Provide(
		fx.Annotate(
			telegramimpl.New,
			fx.As(new(telegram.Client)),
		), fx.Annotate(
			instagramimpl.New,
			fx.As(new(instagram.Client)),
		), fx.Annotate(
			paserimpl.New,
			fx.As(new(parser.Client)),
		),
		fx.Annotate(
			commandimpl.New,
			fx.As(new(command.Client)),
		),
	),
	repositories.Module,
	fx.Invoke(pgx.New),
	fx.Invoke(initMigration),
	fx.Invoke(startHttpServer),
	fx.Invoke(startBot),
)

func initMigration(log logger.Logger, c *config.Config) error {
	if err := goose.SetDialect("pgx"); err != nil {
		return err
	}

	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d sslmode=%s ",
			c.Postgres.Name, c.Postgres.User, c.Postgres.Pass, c.Postgres.Host, c.Postgres.Port, c.Postgres.SslMode,
		),
	)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Error("Failed to close db", "error", err)
		}
	}(db)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	return goose.Up(db, filepath.Join(wd, "migrations"))
}

func startHttpServer(lc fx.Lifecycle, log logger.Logger, cfg *config.Config) {
	router := http.NewServeMux()

	router.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		healthCheckHandler(w, r, log)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info(fmt.Sprintf("Starting server on :%d", cfg.App.Port))
			go func() {
				err := server.ListenAndServe()
				if err != nil {
					log.Error("Server failed to start: %v", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping server")
			return server.Close()
		},
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request, logger logger.Logger) {
	logger.Info("Health check request received", "Method", r.Method, "URL", r.URL.String())
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte("ok")); err != nil {
		logger.Error("Failed to write response", "Error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func startBot(lc fx.Lifecycle, log logger.Logger, tgClient telegram.Client,
	igClient instagram.Client, pClient parser.Client, cmdClient command.Client) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx := context.Background()
			err := igClient.Login()
			if err != nil {
				log.Error("Instagram login error", "Error", err)
				tgClient.SendMessageToUser("Instagram login error:" + err.Error())
			}

			err = pClient.ScheduleParseStories(ctx)
			if err != nil {
				log.Error("Parse stories error", "Error", err)
				tgClient.SendMessageToUser("Parse stories error:" + err.Error())
			}

			//go func() {
			//	for {
			//		if err := command.HandleCommand(); err != nil {
			//			logger.Error("Command error", "Error", err)
			//			telegram.SendMessageToUser("Command error:" + err.Error())
			//		}
			//	}
			//}()

			return nil
		},
	})
}
