package httpx

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter  = otel.Meter("chat-server")
	tracer = otel.Tracer("chat-server")

	requestCounter, _ = meter.Int64Counter(
		"http.requests.total",
		metric.WithDescription("Total HTTP requests"),
	)

	errorCounter, _ = meter.Int64Counter(
		"http.requests.errors",
		metric.WithDescription("Total HTTP error responses"),
	)

	requestDuration, _ = meter.Float64Histogram(
		"http.request.duration.ms",
		metric.WithDescription("HTTP request duration in milliseconds"),
	)
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Metrics() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx, span := tracer.Start(
				r.Context(),
				r.Method+" "+r.URL.Path,
			)
			defer span.End()

			start := time.Now()

			rec := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, r.WithContext(ctx))

			duration := float64(time.Since(start)) / float64(time.Millisecond)

			attrs := metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", rec.status),
			)

			requestCounter.Add(ctx, 1, attrs)

			requestDuration.Record(
				ctx,
				duration,
				attrs,
			)

			if rec.status >= http.StatusBadRequest {
				errorCounter.Add(ctx, 1, attrs)
				span.SetStatus(codes.Error, http.StatusText(rec.status))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}
