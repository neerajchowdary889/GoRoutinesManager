package Common

import (
	"context"

	"github.com/neerajchowdary889/GoRoutinesManager/metrics"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

// resetGlobalState resets the global singleton for testing
func ResetGlobalState() {
	if metrics.IsServerRunning() {
		metrics.StopMetricsServer(context.Background())
	}
	types.Global = nil
}
