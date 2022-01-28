package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	_MyMiddleware "bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/config"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/metrics"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
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
	defer func() {
		// do not need to check for errors
		_ = logger.Sync()
	}()

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

	// Initialize context
	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	// Initialize tracing
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.Server.OtlpAddress),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("shortener-management-api"),
		),
	)
	if err != nil {
		return err
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("shortener-tracer")
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("shutdown tracer provider", zap.Error(err))
		}
		if err := traceExporter.Shutdown(ctx); err != nil {
			logger.Error("shutdown tracing exporter", zap.Error(err))
		}
	}()

	// Initialize metrics
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(cfg.Server.OtlpAddress),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			metricExporter,
		),
		controller.WithExporter(metricExporter),
		controller.WithCollectPeriod(2*time.Second),
		controller.WithResource(res),
	)
	global.SetMeterProvider(pusher)

	if err := pusher.Start(ctx); err != nil {
		return fmt.Errorf("can't start metric's pusher: %w", err)
	}
	defer func() {
		if err := pusher.Stop(ctx); err != nil {
			logger.Error("shutdown pusher", zap.Error(err))
		}
		if err := metricExporter.Shutdown(ctx); err != nil {
			logger.Error("shutdown metric exporter", zap.Error(err))
		}
	}()

	// Echo configure
	e := echo.New()
	middL := _MyMiddleware.InitMiddleware(logger)
	e.Pre(middleware.Rewrite(map[string]string{
		"/api/*": "/$1",
	}))
	e.Use(middL.CORS)
	e.Use(middL.Logger)
	e.Use(middleware.RecoverWithConfig(middleware.DefaultRecoverConfig))
	e.Use(otelecho.Middleware("shortener", otelecho.WithTracerProvider(tp)))
	e.Use(metrics.Middleware(metrics.WithMeterProvider(pusher)))

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
	ur := _URLRepo.NewMongoURLRepository(client, cfg.MongoConfig.Name, logger, tracer)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext, tracer, cfg.Server.URLExpiration)
	uh, err := _URLHttpDelivery.NewURLHandler(uu, authenticator, v, logger, tracer)
	if err != nil {
		return fmt.Errorf("url handler creation failed: %w", err)
	}
	uh.RegisterRoutes(e)

	// Create User API
	usr := _UserRepo.NewMongoUserRepository(client, cfg.MongoConfig.Name, logger, tracer)
	usu := _UserUcase.NewUserUsecase(usr, timeoutContext, tracer)
	ush := _UserHttpDelivery.NewUserHandler(usu, authenticator, v, logger, tracer)
	ush.RegisterRoutes(e)

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
