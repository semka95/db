package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
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

var reqCnt = []metric.InstrumentOption{
	metric.WithDescription("How many HTTP requests processed, partitioned by status code and HTTP method."),
	metric.WithUnit(unit.Dimensionless),
}
var reqDur = []metric.InstrumentOption{
	metric.WithDescription("The HTTP request latencies in milliseconds."),
	metric.WithUnit(unit.Milliseconds),
}
var resSz = []metric.InstrumentOption{
	metric.WithDescription("The HTTP response sizes in bytes."),
	metric.WithUnit(unit.Bytes),
}
var reqSz = []metric.InstrumentOption{
	metric.WithDescription("The HTTP request sizes in bytes."),
	metric.WithUnit(unit.Bytes),
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
		cfg.MeterProvider = global.GetMeterProvider()
	}

	meter := cfg.MeterProvider.Meter(
		meterName,
		metric.WithInstrumentationVersion(contrib.SemVersion()),
	)
	fmt.Println(meter)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Println("I'm in")
			rc, err := meter.NewInt64Counter("requests_total", reqCnt...)
			if err != nil {
				return next(c)
			}
			rd, err := meter.NewFloat64Histogram("request_duration_milliseconds", reqDur...)
			if err != nil {
				return next(c)
			}
			rs, err := meter.NewInt64Histogram("response_size_bytes", resSz...)
			if err != nil {
				return next(c)
			}
			rq, err := meter.NewInt64Histogram("request_size_bytes", reqSz...)
			if err != nil {
				return next(c)
			}

			start := time.Now()
			reqSz := computeApproximateRequestSize(c.Request())
			fmt.Println("request size: ", reqSz)

			if err = next(c); err != nil {
				c.Error(err)
			}

			status := c.Response().Status
			url := c.Path()
			ctx := c.Request().Context()
			fmt.Printf("status: %d, url: %s\n", status, url)

			elapsed := float64(time.Since(start)) / float64(time.Millisecond)
			resSz := c.Response().Size
			fmt.Printf("response size: %d, elapsed: %f\n", resSz, elapsed)

			lbl := []attribute.KeyValue{codeLabel.Int(status), methodLabel.String(c.Request().Method), hostLabel.String(c.Request().Host), urlLabel.String(url)}
			meter.RecordBatch(
				ctx,
				lbl,
				rc.Measurement(1),
				rd.Measurement(elapsed),
				rs.Measurement(resSz),
				rq.Measurement(reqSz),
			)

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
