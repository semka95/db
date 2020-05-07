package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/config"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	_URLHttpDelivery "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	_URLRepo "bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	_URLUcase "bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
)

func main() {
	// Logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Println("can't create logger: ", err)
		os.Exit(1)
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Println("can't close logger: ", err)
			os.Exit(1)
		}
	}()

	if err := run(logger); err != nil {
		logger.Error("shutting down, error: ", zap.Error(err))
		os.Exit(1)
	}
}

func run(logger *zap.Logger) error {
	// Configuration
	cfg, err := config.AppConfig("../../config.yaml", logger)
	if err != nil {
		return err
	}

	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	// Echo configure
	e := echo.New()
	middL := middleware.InitMiddleware(logger)
	e.Use(middL.CORS)
	e.Use(middL.Logger)

	// Start database
	client, err := database.Open(ctx, cfg.MongoConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logger.Error("mongodb client disconnect error: ", zap.Error(err))
		}
	}()

	// Start service
	ur := _URLRepo.NewMongoURLRepository(client, cfg.MongoConfig.Name, logger)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	err = _URLHttpDelivery.NewURLHandler(e, uu, logger)
	if err != nil {
		return fmt.Errorf("url handler creation failed: %w", err)
	}

	// Status check
	database.NewStatusHandler(e, client.Database(cfg.MongoConfig.Name))

	go func() {
		if err := e.Start(cfg.Server.Address); err != nil {
			logger.Error("can't start server: ", zap.Error(err))
		}
	}()

	// Gracefull shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	if err := e.Shutdown(ctx); err != nil {
		return fmt.Errorf("can't shutdownn server: %w", err)
	}

	return nil
}
