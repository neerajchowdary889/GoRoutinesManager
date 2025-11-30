package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Singleton to ensure metrics are registered only once
	once sync.Once

	// metricsInitialized tracks whether metrics have been initialized
	metricsInitialized bool
	metricsLock        sync.RWMutex
)

// Global Manager Metrics
var (
	// GlobalInitialized indicates whether the global manager is initialized
	GlobalInitialized prometheus.Gauge

	// AppManagersTotal tracks the total number of app managers
	AppManagersTotal prometheus.Gauge

	// LocalManagersTotal tracks the total number of local managers across all apps
	LocalManagersTotal prometheus.Gauge

	// GoroutinesTotal tracks the total number of tracked goroutines
	GoroutinesTotal prometheus.Gauge

	// ShutdownTimeoutSeconds tracks the configured shutdown timeout
	ShutdownTimeoutSeconds prometheus.Gauge
)

// App Manager Metrics (with labels)
var (
	// AppLocalManagers tracks the number of local managers per app
	AppLocalManagers *prometheus.GaugeVec

	// AppGoroutines tracks the number of goroutines per app
	AppGoroutines *prometheus.GaugeVec

	// AppInitialized indicates whether an app is initialized
	AppInitialized *prometheus.GaugeVec
)

// Local Manager Metrics (with labels)
var (
	// LocalGoroutines tracks the number of goroutines per local manager
	LocalGoroutines *prometheus.GaugeVec

	// LocalFunctionWaitgroups tracks the number of function wait groups per local manager
	LocalFunctionWaitgroups *prometheus.GaugeVec
)

// Goroutine Metrics (with labels)
var (
	// GoroutinesByFunction tracks the number of goroutines grouped by function
	GoroutinesByFunction *prometheus.GaugeVec

	// GoroutineDuration tracks the duration of goroutines (from start to completion)
	GoroutineDuration *prometheus.HistogramVec

	// GoroutineAge tracks the age of currently running goroutines
	GoroutineAge *prometheus.GaugeVec
)

// Metadata Metrics
var (
	// MaxRoutines tracks the configured maximum routines limit
	MaxRoutines prometheus.Gauge

	// MetricsEnabled indicates whether metrics are enabled
	MetricsEnabled prometheus.Gauge
)

// System Metrics
var (
	// BuildInfo provides build information
	BuildInfo *prometheus.GaugeVec
)

// InitMetrics initializes and registers all Prometheus metrics
// This function is safe to call multiple times (uses sync.Once)
func InitMetrics() {
	once.Do(func() {
		initGlobalMetrics()
		initAppMetrics()
		initLocalMetrics()
		initGoroutineMetrics()
		initMetadataMetrics()
		initSystemMetrics()

		metricsLock.Lock()
		metricsInitialized = true
		metricsLock.Unlock()
	})
}

// IsInitialized returns whether metrics have been initialized
func IsInitialized() bool {
	metricsLock.RLock()
	defer metricsLock.RUnlock()
	return metricsInitialized
}

func initGlobalMetrics() {
	GlobalInitialized = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "global",
		Name:      "initialized",
		Help:      "Whether the global manager is initialized (1 = yes, 0 = no)",
	})

	AppManagersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "global",
		Name:      "app_managers_total",
		Help:      "Total number of app managers",
	})

	LocalManagersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "global",
		Name:      "local_managers_total",
		Help:      "Total number of local managers across all apps",
	})

	GoroutinesTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "global",
		Name:      "goroutines_total",
		Help:      "Total number of tracked goroutines",
	})

	ShutdownTimeoutSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "global",
		Name:      "shutdown_timeout_seconds",
		Help:      "Configured shutdown timeout in seconds",
	})
}

func initAppMetrics() {
	AppLocalManagers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "app",
			Name:      "local_managers",
			Help:      "Number of local managers per app",
		},
		[]string{"app_name"},
	)

	AppGoroutines = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "app",
			Name:      "goroutines",
			Help:      "Number of goroutines per app",
		},
		[]string{"app_name"},
	)

	AppInitialized = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "app",
			Name:      "initialized",
			Help:      "Whether an app is initialized (1 = yes, 0 = no)",
		},
		[]string{"app_name"},
	)
}

func initLocalMetrics() {
	LocalGoroutines = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "local",
			Name:      "goroutines",
			Help:      "Number of goroutines per local manager",
		},
		[]string{"app_name", "local_name"},
	)

	LocalFunctionWaitgroups = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "local",
			Name:      "function_waitgroups",
			Help:      "Number of function wait groups per local manager",
		},
		[]string{"app_name", "local_name"},
	)
}

func initGoroutineMetrics() {
	GoroutinesByFunction = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "goroutine",
			Name:      "by_function",
			Help:      "Number of goroutines grouped by function",
		},
		[]string{"app_name", "local_name", "function_name"},
	)

	GoroutineDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "goroutine_manager",
			Subsystem: "goroutine",
			Name:      "duration_seconds",
			Help:      "Duration of goroutines from start to completion",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		[]string{"app_name", "local_name", "function_name"},
	)

	GoroutineAge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "goroutine",
			Name:      "age_seconds",
			Help:      "Age of currently running goroutines in seconds",
		},
		[]string{"app_name", "local_name", "function_name", "routine_id"},
	)
}

func initMetadataMetrics() {
	MaxRoutines = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "metadata",
		Name:      "max_routines",
		Help:      "Configured maximum routines limit (0 = unlimited)",
	})

	MetricsEnabled = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "goroutine_manager",
		Subsystem: "metadata",
		Name:      "enabled",
		Help:      "Whether metrics collection is enabled (1 = yes, 0 = no)",
	})
}

func initSystemMetrics() {
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "goroutine_manager",
			Subsystem: "system",
			Name:      "build_info",
			Help:      "Build information for the goroutine manager",
		},
		[]string{"version", "go_version"},
	)
}

// RecordGoroutineCompletion records the completion of a goroutine
// This should be called when a goroutine finishes execution
func RecordGoroutineCompletion(appName, localName, functionName string, startTime int64) {
	if !IsInitialized() {
		return
	}

	duration := time.Since(time.Unix(0, startTime)).Seconds()
	GoroutineDuration.WithLabelValues(appName, localName, functionName).Observe(duration)
}

// UpdateGoroutineAge updates the age metric for a specific goroutine
func UpdateGoroutineAge(appName, localName, functionName, routineID string, startTime int64) {
	if !IsInitialized() {
		return
	}

	age := time.Since(time.Unix(0, startTime)).Seconds()
	GoroutineAge.WithLabelValues(appName, localName, functionName, routineID).Set(age)
}

// RemoveGoroutineAge removes the age metric for a specific goroutine
func RemoveGoroutineAge(appName, localName, functionName, routineID string) {
	if !IsInitialized() {
		return
	}

	GoroutineAge.DeleteLabelValues(appName, localName, functionName, routineID)
}
