package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/semka95/shortener/backend/cmd"
	"github.com/semka95/shortener/backend/metrics"
	_MyMiddleware "github.com/semka95/shortener/backend/middleware"
	"github.com/semka95/shortener/backend/store"
	_URLHttpDelivery "github.com/semka95/shortener/backend/url/delivery/http"
	_URLRepo "github.com/semka95/shortener/backend/url/repository"
	_URLUcase "github.com/semka95/shortener/backend/url/usecase"
	_UserHttpDelivery "github.com/semka95/shortener/backend/user/delivery/http"
	_UserRepo "github.com/semka95/shortener/backend/user/repository"
	_UserUcase "github.com/semka95/shortener/backend/user/usecase"
	"github.com/semka95/shortener/backend/web"
	"github.com/semka95/shortener/backend/web/auth"
)

func main() {
	// Logging
	logger, err := zap.NewDevelopment(zap.AddCaller())
	if err != nil {
		log.Println("can't create logger: ", err)
		return
	}
	defer func() {
		// do not need to check for errors
		_ = logger.Sync()
	}()

	if err := run(logger); err != nil {
		logger.Error("shutting down, error: ", zap.Error(err))
	}
}

func run(logger *zap.Logger) error {
	// Configuration
	configPath, ok := os.LookupEnv("SHORTENER_CONFIG")
	if !ok {
		return fmt.Errorf("SHORTENER_CONFIG environment variable is not specified")
	}
	logger.Info("Config path", zap.String(configPath, configPath))
	cfg, err := cmd.AppConfig(configPath, logger)
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
	ctx, cancel := context.WithCancel(context.Background())
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
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // dev env only
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("shortener-tracer")
	defer func() {
		if err = tp.Shutdown(ctx); err != nil {
			logger.Error("shutdown tracer provider", zap.Error(err))
		}
		if err = traceExporter.Shutdown(ctx); err != nil {
			logger.Error("shutdown tracing exporter", zap.Error(err))
		}
	}()

	// Initialize metrics
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(cfg.Server.OtlpAddress),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(10*time.Second))),
		metric.WithResource(res),
	)
	global.SetMeterProvider(meterProvider)

	defer func() {
		if err = meterProvider.Shutdown(ctx); err != nil {
			logger.Error("shutdown meter provider", zap.Error(err))
		}
		if err = metricExporter.Shutdown(ctx); err != nil {
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
	e.Use(metrics.Middleware(metrics.WithMeterProvider(meterProvider)))

	// Create database connection
	client, err := store.Open(ctx, cfg.MongoConfig, logger)
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
	store.NewStatusHandler(e, client.Database(cfg.MongoConfig.Name))

	go func() {
		if err := e.Start(cfg.Server.Address); err != nil {
			logger.Error("can't start server: ", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancelSrv := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelSrv()
	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("can't shutdownn server: %w", err)
	}

	return nil
}

func createAuth(privateKeyFile, keyID, algorithm string) (*auth.Authenticator, error) {
	keyContents, err := os.ReadFile(privateKeyFile)
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
