package Global

import (
	"errors"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"
	"github.com/neerajchowdary889/GoRoutinesManager/metrics"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

const (
	SET_METRICS_URL      = "SET_METRICS_URL"
	SET_SHUTDOWN_TIMEOUT = "SET_SHUTDOWN_TIMEOUT"
	SET_MAX_ROUTINES     = "SET_MAX_ROUTINES"
)

type metricsConfig struct {
	Enabled bool
	URL     string
}

func (GM *GlobalManager) UpdateGlobalMetadata(flag string, value interface{}) (*types.Metadata, error) {
	// Get the global manager first
	g, err := types.GetGlobalManager()
	if err != nil {
		return nil, err
	}

	metadata := g.NewMetadata()

	switch flag {
	case SET_METRICS_URL:
		// Accept several input shapes:
		//  - string -> URL (enabled = true)
		//  - metricsConfig
		//  - []interface{}{bool, string} or [2]interface{}{bool, string}
		var enabled bool
		var url string
		switch v := value.(type) {
		case string:
			enabled = true
			url = v
			metadata.SetMetrics(enabled, url)
		case metricsConfig:
			enabled = v.Enabled
			url = v.URL
			metadata.SetMetrics(v.Enabled, v.URL)
		case *metricsConfig:
			enabled = v.Enabled
			url = v.URL
			metadata.SetMetrics(v.Enabled, v.URL)
		case []interface{}:
			if len(v) != 2 {
				return nil, errors.New("metrics: expected slice of length 2: [enabled(bool), url(string)]")
			}
			enabledVal, ok1 := v[0].(bool)
			urlVal, ok2 := v[1].(string)
			if !ok1 || !ok2 {
				return nil, errors.New("metrics: expected [bool, string] in slice")
			}
			enabled = enabledVal
			url = urlVal
			metadata.SetMetrics(enabledVal, urlVal)
		case [2]interface{}:
			enabledVal, ok1 := v[0].(bool)
			urlVal, ok2 := v[1].(string)
			if !ok1 || !ok2 {
				return nil, errors.New("metrics: expected [bool, string] array")
			}
			enabled = enabledVal
			url = urlVal
			metadata.SetMetrics(enabledVal, urlVal)
		default:
			return nil, errors.New("metrics: unsupported value type; expected string, metricsConfig, or [bool,string]")
		}

		// Initialize metrics if enabled
		if enabled {
			metrics.InitMetrics()
			if url != "" {
				// Start the metrics server if a URL is provided
				if err := metrics.StartMetricsServer(url, 5*time.Second); err != nil {
					// If server is already running, we might want to restart it or ignore
					// For now, we'll return the error if it fails to start
					return nil, err
				}
			} else {
				// Start the collector (metrics will be collected periodically)
				// The collector runs in the background and updates metrics every 5 seconds by default
				metrics.StartCollector(5 * time.Second)
			}
		} else {
			// Stop the collector if metrics are disabled
			metrics.StopCollector()
			// Also stop the metrics server if it's running
			// Get the parent context for this - dont create a new context
			// spawn a child context from global context
			ctx, cancel := Context.SpawnChild(g.Ctx)
			metrics.StopMetricsServer(ctx)
			cancel()
		}

	case SET_SHUTDOWN_TIMEOUT:
		switch t := value.(type) {
		case time.Duration:
			metadata.SetShutdownTimeout(t)
		case *time.Duration:
			metadata.SetShutdownTimeout(*t)
		default:
			return nil, errors.New("shutdown timeout: expected time.Duration")
		}

	case SET_MAX_ROUTINES:
		switch n := value.(type) {
		case int:
			metadata.SetMaxRoutines(n)
		case int32:
			metadata.SetMaxRoutines(int(n))
		case int64:
			metadata.SetMaxRoutines(int(n))
		case *int:
			metadata.SetMaxRoutines(*n)
		default:
			return nil, errors.New("max routines: expected integer type")
		}

	default:
		return nil, errors.New("unknown update flag")
	}

	return metadata, nil
}

func (GM *GlobalManager) GetGlobalMetadata() (*types.Metadata, error) {
	g, err := types.GetGlobalManager()
	if err != nil {
		return nil, err
	}
	return g.GetMetadata(), nil
}
