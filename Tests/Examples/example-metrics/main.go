package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/App"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Global"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Local"
	"github.com/neerajchowdary889/GoRoutinesManager/metrics"
)

// Service represents a complete service using GoRoutinesManager with metrics
type Service struct {
	globalMgr     Interface.GlobalGoroutineManagerInterface
	apiApp        Interface.AppGoroutineManagerInterface
	workerApp     Interface.AppGoroutineManagerInterface
	httpServer    *http.Server
	metricsPort   string
	metricsServer *http.Server
}

// NewService creates and initializes a new service with metrics enabled
func NewService() (*Service, error) {
	// 1. Initialize Global Manager
	globalMgr := Global.NewGlobalManager()
	_, err := globalMgr.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize global manager: %w", err)
	}

	// 2. Enable metrics with 2 second update interval
	metricsPort := ":19090"
	_, err = globalMgr.UpdateMetadata(Global.SET_METRICS_URL, []interface{}{true, "", 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to enable metrics: %w", err)
	}

	// 3. Create API Server App
	apiApp := App.NewAppManager("api-server")
	if _, err := apiApp.CreateApp(); err != nil {
		return nil, fmt.Errorf("failed to create API app: %w", err)
	}

	// 4. Create Worker Pool App
	workerApp := App.NewAppManager("worker-pool")
	if _, err := workerApp.CreateApp(); err != nil {
		return nil, fmt.Errorf("failed to create worker app: %w", err)
	}

	return &Service{
		globalMgr:   globalMgr,
		apiApp:      apiApp,
		workerApp:   workerApp,
		metricsPort: metricsPort,
	}, nil
}

// Start initializes and starts all service components
func (s *Service) Start() error {
	log.Println("Starting service with metrics...")

	// Start metrics collector
	metrics.StartCollector()
	log.Println("✓ Metrics collector started")

	// Setup metrics server
	if err := s.setupMetricsServer(); err != nil {
		return fmt.Errorf("failed to setup metrics server: %w", err)
	}
	log.Println("✓ Metrics server setup complete")

	// Setup API Server
	if err := s.setupAPIServer(); err != nil {
		return fmt.Errorf("failed to setup API server: %w", err)
	}
	log.Println("✓ API server setup complete")

	// Setup Background Workers
	if err := s.setupWorkers(); err != nil {
		return fmt.Errorf("failed to setup workers: %w", err)
	}
	log.Println("✓ Background workers setup complete")

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on :8080")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("✓ Service started successfully")
	return nil
}

// setupMetricsServer configures the metrics HTTP server
func (s *Service) setupMetricsServer() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.GetMetricsHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.metricsServer = &http.Server{
		Addr:    s.metricsPort,
		Handler: mux,
	}

	// Start metrics server in background
	go func() {
		log.Printf("Starting metrics server on %s", s.metricsPort)
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// setupAPIServer configures the HTTP API server
func (s *Service) setupAPIServer() error {
	// Create local manager for API handlers
	apiLocal := Local.NewLocalManager("api-server", "handlers")
	if _, err := apiLocal.CreateLocal("handlers"); err != nil {
		return err
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/api/status", s.statusHandler)
	mux.HandleFunc("/api/metrics", s.metricsHandler)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Spawn goroutine to handle server shutdown
	apiLocal.Go("http-server", func(ctx context.Context) error {
		// Wait for context cancellation (shutdown signal)
		<-ctx.Done()

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Println("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
			return err
		}
		log.Println("✓ HTTP server shut down gracefully")
		return nil
	}, Local.AddToWaitGroup("http-server"))

	return nil
}

// setupWorkers configures background workers
func (s *Service) setupWorkers() error {
	// Create local manager for workers
	workerLocal := Local.NewLocalManager("worker-pool", "jobs")
	if _, err := workerLocal.CreateLocal("jobs"); err != nil {
		return err
	}

	// Spawn multiple worker goroutines
	for i := 1; i <= 5; i++ {
		workerID := i
		workerLocal.Go("job-processor", func(ctx context.Context) error {
			log.Printf("Worker %d started", workerID)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d shutting down...", workerID)
					return nil
				case <-ticker.C:
					// Simulate work
					log.Printf("Worker %d processing job...", workerID)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}, Local.AddToWaitGroup("job-processor"))
	}

	// Spawn a periodic task worker
	workerLocal.Go("periodic-task", func(ctx context.Context) error {
		log.Println("Periodic task worker started")
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Periodic task worker shutting down...")
				return nil
			case <-ticker.C:
				log.Println("Executing periodic task...")
				// Simulate periodic work
				time.Sleep(500 * time.Millisecond)
			}
		}
	}, Local.AddToWaitGroup("periodic-task"))

	// Spawn a worker with timeout protection
	workerLocal.Go("timeout-worker", func(ctx context.Context) error {
		log.Println("Timeout-protected worker started")
		for {
			select {
			case <-ctx.Done():
				log.Println("Timeout worker shutting down...")
				return nil
			default:
				// Long-running task that respects context cancellation
				time.Sleep(1 * time.Second)
			}
		}
	}, Local.WithTimeout(2*time.Minute), Local.AddToWaitGroup("timeout-worker"))

	return nil
}

// HTTP Handlers

func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Service) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get service statistics
	apiCount := s.apiApp.GetGoroutineCount()
	workerCount := s.workerApp.GetGoroutineCount()
	totalCount := s.globalMgr.GetGoroutineCount()

	status := map[string]interface{}{
		"service": "GoRoutinesManager Example Service with Metrics",
		"status":  "running",
		"goroutines": map[string]int{
			"api":    apiCount,
			"worker": workerCount,
			"total":  totalCount,
		},
		"apps": map[string]int{
			"total": s.globalMgr.GetAppManagerCount(),
		},
		"metrics": map[string]string{
			"endpoint":   fmt.Sprintf("http://localhost%s/metrics", s.metricsPort),
			"prometheus": "http://localhost:9090",
			"grafana":    "http://localhost:3000",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (s *Service) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get detailed metrics
	metadata, err := s.globalMgr.GetMetadata()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metrics := map[string]interface{}{
		"metrics_enabled":      metadata.GetMetrics(),
		"max_routines":         metadata.GetMaxRoutines(),
		"shutdown_timeout":     metadata.GetShutdownTimeout().String(),
		"update_interval":      metadata.GetUpdateInterval().String(),
		"total_goroutines":     s.globalMgr.GetGoroutineCount(),
		"total_apps":           s.globalMgr.GetAppManagerCount(),
		"total_local_managers": s.globalMgr.GetLocalManagerCount(),
		"metrics_endpoint":     fmt.Sprintf("http://localhost%s/metrics", s.metricsPort),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown() error {
	log.Println("Initiating graceful shutdown...")

	// Stop metrics collector
	metrics.StopCollector()
	log.Println("✓ Metrics collector stopped")

	// Shutdown metrics server
	if s.metricsServer != nil {
		serverCtx, serverCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer serverCancel()
		if err := s.metricsServer.Shutdown(serverCtx); err != nil {
			log.Printf("Metrics server shutdown warning: %v", err)
		} else {
			log.Println("✓ Metrics server shut down")
		}
	}

	// Shutdown global manager (will shutdown all apps and goroutines)
	if err := s.globalMgr.Shutdown(true); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	log.Println("✓ Service shut down successfully")
	return nil
}

func main() {
	// Create and start service
	service, err := NewService()
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Get global context to wait for shutdown signal
	// The global context is automatically set up with signal handling (SIGINT/SIGTERM)
	// When a signal is received, the context will be cancelled, triggering shutdown
	globalCtx := Context.GetGlobalContext().Get()

	log.Println("\n" + strings.Repeat("=", 80))
	log.Println("Service is running with metrics enabled!")
	log.Println(strings.Repeat("=", 80))
	log.Println("API available at: http://localhost:8080")
	log.Println("  - GET /health - Health check")
	log.Println("  - GET /api/status - Service status")
	log.Println("  - GET /api/metrics - Metrics info")
	log.Println("\n📊 Metrics available at: http://localhost:19090/metrics")
	log.Println("📊 Prometheus should scrape: http://localhost:19090/metrics")
	log.Println("📊 Prometheus UI: http://localhost:9090")
	log.Println("📊 Grafana UI: http://localhost:3000 (admin/admin)")
	log.Println("\n💡 To start Prometheus and Grafana:")
	log.Println("   cd Tests/example-metrics && docker-compose up -d")
	log.Println("\n⚠️  Press Ctrl+C to shutdown gracefully...")

	// Wait for shutdown signal (SIGINT/SIGTERM)
	// The global context will be cancelled when signals are received
	<-globalCtx.Done()

	// Perform graceful shutdown
	log.Println("\nShutdown signal received, initiating graceful shutdown...")
	if err := service.Shutdown(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
}
