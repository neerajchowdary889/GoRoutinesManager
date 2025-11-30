package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// metricsServer holds the HTTP server instance
	metricsServer *http.Server
	serverLock    sync.Mutex

	// defaultCollector is the default metrics collector
	defaultCollector *Collector
	collectorLock    sync.Mutex
)

// StartMetricsServer starts an HTTP server to expose Prometheus metrics
// addr is the address to listen on (e.g., ":9090" or "localhost:9090")
// updateInterval is how often to collect metrics (0 = use default of 5 seconds)
func StartMetricsServer(addr string, updateInterval time.Duration) error {
	serverLock.Lock()
	defer serverLock.Unlock()

	if metricsServer != nil {
		return fmt.Errorf("metrics server is already running")
	}

	// Initialize metrics if not already done
	InitMetrics()

	// Create and start the collector
	collectorLock.Lock()
	if defaultCollector == nil {
		defaultCollector = NewCollector(updateInterval)
		defaultCollector.Start()
	}
	collectorLock.Unlock()

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add a root endpoint with information
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>GoRoutinesManager Metrics</title>
</head>
<body>
    <h1>GoRoutinesManager Metrics Exporter</h1>
    <p>Prometheus metrics are available at <a href="/metrics">/metrics</a></p>
    <p>Health check is available at <a href="/health">/health</a></p>
</body>
</html>
		`))
	})

	metricsServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine
	go func(server *http.Server) {
		log.Printf("Starting metrics server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}(metricsServer)

	return nil
}

// StopMetricsServer gracefully stops the metrics HTTP server
func StopMetricsServer(ctx context.Context) error {
	serverLock.Lock()
	defer serverLock.Unlock()

	if metricsServer == nil {
		return fmt.Errorf("metrics server is not running")
	}

	// Stop the collector
	collectorLock.Lock()
	if defaultCollector != nil {
		defaultCollector.Stop()
		defaultCollector = nil
	}
	collectorLock.Unlock()

	// Shutdown the HTTP server
	err := metricsServer.Shutdown(ctx)
	metricsServer = nil

	return err
}

// GetMetricsHandler returns the HTTP handler for Prometheus metrics
// This allows users to integrate metrics into their own HTTP server
func GetMetricsHandler() http.Handler {
	// Initialize metrics if not already done
	InitMetrics()

	return promhttp.Handler()
}

// StartCollector starts the metrics collector without starting an HTTP server
// This is useful when you want to integrate metrics into an existing HTTP server
// updateInterval is how often to collect metrics (0 = use default of 5 seconds)
func StartCollector(updateInterval time.Duration) {
	collectorLock.Lock()
	defer collectorLock.Unlock()

	// Initialize metrics if not already done
	InitMetrics()

	if defaultCollector == nil {
		defaultCollector = NewCollector(updateInterval)
		defaultCollector.Start()
	}
}

// StopCollector stops the metrics collector
func StopCollector() {
	collectorLock.Lock()
	defer collectorLock.Unlock()

	if defaultCollector != nil {
		defaultCollector.Stop()
		defaultCollector = nil
	}
}

// IsServerRunning returns whether the metrics server is currently running
func IsServerRunning() bool {
	serverLock.Lock()
	defer serverLock.Unlock()
	return metricsServer != nil
}

// IsCollectorRunning returns whether the metrics collector is currently running
func IsCollectorRunning() bool {
	collectorLock.Lock()
	defer collectorLock.Unlock()
	return defaultCollector != nil && defaultCollector.running
}
