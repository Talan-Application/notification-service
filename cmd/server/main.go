package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/internal/config"
	"github.com/Talan-Application/notification-service/internal/consumer"
	"github.com/Talan-Application/notification-service/internal/domain"
	"github.com/Talan-Application/notification-service/internal/handler"
	"github.com/Talan-Application/notification-service/internal/repository/postgres"
	consolesender "github.com/Talan-Application/notification-service/internal/sender/console"
	smtpsender "github.com/Talan-Application/notification-service/internal/sender/smtp"
	"github.com/Talan-Application/notification-service/internal/sender"
	"github.com/Talan-Application/notification-service/pkg/logger"
	"github.com/Talan-Application/notification-service/pkg/rabbitmq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	zapLog := logger.New(cfg.App.Env)
	defer zapLog.Sync() //nolint:errcheck

	// ── Database ────────────────────────────────────────────────────────────
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		zapLog.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	// ── RabbitMQ ────────────────────────────────────────────────────────────
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL, zapLog)
	if err != nil {
		zapLog.Fatal("rabbitmq connection failed", zap.Error(err))
	}
	defer rabbitConn.Close()

	// ── Email sender ────────────────────────────────────────────────────────
	var emailSender sender.EmailSender
	if cfg.App.SenderType == "smtp" {
		emailSender = smtpsender.NewSender(cfg.SMTP)
	} else {
		emailSender = consolesender.NewSender(zapLog)
	}

	// ── Event handlers ──────────────────────────────────────────────────────
	handlers := map[domain.EventType]handler.EventHandler{
		domain.EventUserRegistered: handler.NewUserRegisteredHandler(emailSender, zapLog),
		domain.EventPasswordReset:  handler.NewPasswordResetHandler(emailSender, zapLog),
	}

	// ── Consumer ─────────────────────────────────────────────────────────────
	idempotencyRepo := postgres.NewIdempotencyRepository(db)
	notifConsumer := consumer.NewNotificationConsumer(handlers, idempotencyRepo, zapLog)

	mqConsumer, err := consumer.New(rabbitConn, zapLog)
	if err != nil {
		zapLog.Fatal("failed to create consumer", zap.Error(err))
	}
	defer mqConsumer.Close()

	// ── Lifecycle ────────────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- mqConsumer.Consume(ctx, notifConsumer.Handle)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	select {
	case sig := <-quit:
		zapLog.Info("shutdown signal received", zap.String("signal", sig.String()))
		// Cancel context — Consumer.Consume drains in-flight messages before returning.
		cancel()
		select {
		case <-consumerDone:
			zapLog.Info("graceful shutdown complete")
		case <-shutdownCtx.Done():
			zapLog.Warn("shutdown timed out, forcing exit")
		}

	case err := <-consumerDone:
		// Consumer exited on its own (channel closed, fatal error, etc.)
		cancel()
		if err != nil {
			zapLog.Error("consumer exited unexpectedly", zap.Error(err))
			os.Exit(1)
		}
	}
}
