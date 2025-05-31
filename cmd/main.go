package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/app"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

func main() {
	log := logger.New(logger.Opts{})

	app := fx.New(
		fx.Logger(log),
		app.Module,
	)

	// Start the application
	if err := app.Start(context.Background()); err != nil {
		log.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully shutdown the application
	if err := app.Stop(context.Background()); err != nil {
		log.Error("Failed to stop application", "error", err)
		os.Exit(1)
	}
}
