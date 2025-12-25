package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const (
	defaultPort        = "8080"
	defaultServiceName = "cloud-run-service-go"
)

func main() {
	// Configure structured logging with zerolog
	configureLogging()

	// Get configuration from environment
	port := getEnv("PORT", defaultPort)
	serviceName := getEnv("SERVICE_NAME", defaultServiceName)
	
	log.Info().
		Str("service", serviceName).
		Str("port", port).
		Msg("Starting service")

	// Initialize OpenTelemetry
	ctx := context.Background()
	shutdown, err := initTracer(ctx, serviceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize tracer")
		// Continue without tracing rather than failing
	}
	if shutdown != nil {
		defer func() {
			if err := shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to shutdown tracer")
			}
		}()
	}

	// Create router with chi
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	
	// Add OpenTelemetry middleware if tracing is enabled
	if shutdown != nil {
		r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))
	}

	// Health check endpoints
	r.Get("/health", healthHandler)
	r.Get("/readiness", readinessHandler)
	r.Get("/liveness", livenessHandler)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", apiRootHandler)
		r.Get("/hello", helloHandler)
		r.Post("/echo", echoHandler)
	})

	// Root endpoint
	r.Get("/", rootHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// configureLogging sets up zerolog with appropriate configuration
func configureLogging() {
	// Use JSON logging in production, pretty logging in development
	if getEnv("ENVIRONMENT", "production") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	// Set log level
	level := getEnv("LOG_LEVEL", "info")
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// loggerMiddleware creates a structured logging middleware using zerolog
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		defer func() {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).
				Str("request_id", middleware.GetReqID(r.Context())).
				Msg("HTTP request")
		}()
		
		next.ServeHTTP(ww, r)
	})
}

// initTracer initializes OpenTelemetry tracing
func initTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Check if tracing is enabled
	if getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "") == "" {
		log.Info().Msg("OpenTelemetry tracing not configured (OTEL_EXPORTER_OTLP_ENDPOINT not set)")
		return nil, nil
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	log.Info().Msg("OpenTelemetry tracing initialized")

	return tp.Shutdown, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Health check handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	// Add any readiness checks here (database connectivity, etc.)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive"}`))
}

// API handlers
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Cloud Run Service Go","version":"1.0.0"}`))
}

func apiRootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"API v1","endpoints":["/api/v1/hello","/api/v1/echo"]}`))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	
	log.Debug().Str("name", name).Msg("Hello endpoint called")
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"Hello, %s!"}`, name)
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Simply echo back the request body
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.WriteHeader(http.StatusOK)
	
	// Copy request body to response
	if r.Body != nil {
		defer r.Body.Close()
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}
}
