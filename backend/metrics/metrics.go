package metrics

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
)

const meterName = "test-meter"

// config is used to configure the mux middleware.
type config struct {
	MeterProvider metric.MeterProvider
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithMeterProvider option sets metric provider. If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return func(cfg *config) {
		cfg.MeterProvider = provider
	}
}

var reqCnt = []instrument.Int64Option{
	instrument.WithDescription("How many HTTP requests processed, partitioned by status code and HTTP method."),
	instrument.WithUnit(unit.Dimensionless),
}
var reqDur = []instrument.Float64Option{
	instrument.WithDescription("The HTTP request latencies in milliseconds."),
	instrument.WithUnit(unit.Milliseconds),
}
var resSz = []instrument.Int64Option{
	instrument.WithDescription("The HTTP response sizes in bytes."),
	instrument.WithUnit(unit.Bytes),
}
var reqSz = []instrument.Int64Option{
	instrument.WithDescription("The HTTP request sizes in bytes."),
	instrument.WithUnit(unit.Bytes),
}

var codeLabel = attribute.Key("code")
var methodLabel = attribute.Key("method")
var hostLabel = attribute.Key("host")
var urlLabel = attribute.Key("url")

// Middleware represents metric middleware
func Middleware(opts ...Option) echo.MiddlewareFunc {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = global.MeterProvider()
	}

	meter := cfg.MeterProvider.Meter(
		meterName,
		metric.WithInstrumentationVersion(contrib.SemVersion()),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rc, err := meter.Int64Counter("requests_total", reqCnt...)
			if err != nil {
				return next(c)
			}
			rd, err := meter.Float64Histogram("request_duration_milliseconds", reqDur...)
			if err != nil {
				return next(c)
			}
			rs, err := meter.Int64Histogram("response_size_bytes", resSz...)
			if err != nil {
				return next(c)
			}
			rq, err := meter.Int64Histogram("request_size_bytes", reqSz...)
			if err != nil {
				return next(c)
			}

			start := time.Now()
			reqSz := computeApproximateRequestSize(c.Request())

			if err = next(c); err != nil {
				c.Error(err)
			}

			status := c.Response().Status
			url := c.Path()
			ctx := c.Request().Context()

			elapsed := float64(time.Since(start)) / float64(time.Millisecond)
			resSz := c.Response().Size

			lbl := []attribute.KeyValue{codeLabel.Int(status), methodLabel.String(c.Request().Method), hostLabel.String(c.Request().Host), urlLabel.String(url)}

			rc.Add(ctx, 1, lbl...)
			rd.Record(ctx, elapsed, lbl...)
			rs.Record(ctx, resSz, lbl...)
			rq.Record(ctx, reqSz, lbl...)

			return nil
		}
	}
}

func computeApproximateRequestSize(r *http.Request) int64 {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return int64(s)
}
