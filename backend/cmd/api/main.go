package main

import (
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"context"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	_MyMiddleware "bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/config"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	_URLHttpDelivery "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	_URLRepo "bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	_URLUcase "bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
	_UserHttpDelivery "bitbucket.org/dbproject_ivt/db/backend/internal/user/delivery/http"
	_UserRepo "bitbucket.org/dbproject_ivt/db/backend/internal/user/repository"
	_UserUcase "bitbucket.org/dbproject_ivt/db/backend/internal/user/usecase"
)

func main() {
	// Logging
	logger, err := zap.NewDevelopment(zap.AddCaller())
	if err != nil {
		log.Println("can't create logger: ", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if err := run(logger); err != nil {
		logger.Error("shutting down, error: ", zap.Error(err))
		os.Exit(1)
	}
}

func run(logger *zap.Logger) error {
	// Configuration
	configPath, ok := os.LookupEnv("SHORTENER_CONFIG")
	if !ok {
		return fmt.Errorf("SHORTENER_CONFIG environment variable is not specified")
	}
	logger.Info("Config path", zap.String(configPath, configPath))
	cfg, err := config.AppConfig(configPath, logger)
	if err != nil {
		return err
	}

	// Initialize authentication support
	authenticator, err := createAuth(cfg.Auth.PrivateKeyFile, cfg.Auth.KeyID, cfg.Auth.Algorithm)
	if err != nil {
		return err
	}

	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	// Echo configure
	e := echo.New()
	middL := _MyMiddleware.InitMiddleware(logger)
	e.Pre(middleware.Rewrite(map[string]string{
		"/api/*": "/$1",
	}))
	e.Use(middL.CORS)
	e.Use(middL.Logger)
	e.Use(middleware.RecoverWithConfig(middleware.DefaultRecoverConfig))

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

	// Initialize validator
	v, err := web.NewAppValidator()
	if err != nil {
		return err
	}
	e.Validator = v

	// Create URL API
	ur := _URLRepo.NewMongoURLRepository(client, cfg.MongoConfig.Name, logger)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	err = _URLHttpDelivery.NewURLHandler(e, uu, authenticator, v, logger)
	if err != nil {
		return fmt.Errorf("url handler creation failed: %w", err)
	}

	// Create User API
	usr := _UserRepo.NewMongoUserRepository(client, cfg.MongoConfig.Name, logger)
	usu := _UserUcase.NewUserUsecase(usr, timeoutContext)
	err = _UserHttpDelivery.NewUserHandler(e, usu, authenticator, v, logger)
	if err != nil {
		return fmt.Errorf("user handler creation failed: %w", err)
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
	shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("can't shutdownn server: %w", err)
	}

	return nil
}

func createAuth(privateKeyFile, keyID, algorithm string) (*auth.Authenticator, error) {
	keyContents, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("can't read auth private key: %w", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyContents)
	if err != nil {
		return nil, fmt.Errorf("can't parse auth private key: %w", err)
	}

	public := auth.NewSimpleKeyLookupFunc(keyID, key.Public().(*rsa.PublicKey))

	return auth.NewAuthenticator(key, keyID, algorithm, public)
}
