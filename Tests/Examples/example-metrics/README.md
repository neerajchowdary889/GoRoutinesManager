# Example Service with Metrics

This example demonstrates how to use GoRoutinesManager with full metrics integration, including Prometheus and Grafana dashboards.

## Features

- **Complete Service Example**: HTTP API server with background workers
- **Metrics Integration**: Prometheus metrics exposed on `/metrics` endpoint
- **Observability**: Full dashboard setup with Grafana
- **Graceful Shutdown**: Proper cleanup of all resources

## Architecture

The example service includes:

1. **API Server App** (`api-server`)
   - HTTP handlers for health checks and status
   - HTTP server with graceful shutdown

2. **Worker Pool App** (`worker-pool`)
   - 5 job processor workers
   - 1 periodic task worker
   - 1 timeout-protected worker

3. **Metrics Server**
   - Exposes Prometheus metrics on port `:19090`
   - Health check endpoint

## Quick Start

### Option 1: Run with Docker Compose (Recommended)

This is the easiest way to run the example with Prometheus and Grafana:

```bash
cd Tests/example-metrics
./run-with-prometheus.sh
```

This script will:
1. Start Prometheus and Grafana in Docker
2. Wait for them to be ready
3. Run the example service
4. Keep services running until you press Enter

### Option 2: Manual Setup

**Step 1: Start Prometheus and Grafana**

```bash
cd Tests/example-metrics
docker-compose up -d
```

**Step 2: Run the Example Service**

In another terminal:

```bash
cd Tests/example-metrics
go run main.go
```

**Step 3: Access the Services**

- **API Server**: http://localhost:8080
  - `GET /health` - Health check
  - `GET /api/status` - Service status
  - `GET /api/metrics` - Metrics information

- **Metrics Endpoint**: http://localhost:19090/metrics
  - Prometheus-compatible metrics

- **Prometheus UI**: http://localhost:9090
  - Query metrics, view targets, create graphs

- **Grafana UI**: http://localhost:3000
  - Username: `admin`
  - Password: `admin`
  - View dashboards and create visualizations

**Step 4: Stop Services**

```bash
docker-compose down
```

## Service Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "time": "2025-12-02T10:30:00Z"
}
```

### Service Status
```bash
curl http://localhost:8080/api/status
```

Response:
```json
{
  "service": "GoRoutinesManager Example Service with Metrics",
  "status": "running",
  "goroutines": {
    "api": 1,
    "worker": 7,
    "total": 8
  },
  "apps": {
    "total": 2
  },
  "metrics": {
    "endpoint": "http://localhost:19090/metrics",
    "prometheus": "http://localhost:9090",
    "grafana": "http://localhost:3000"
  },
  "timestamp": "2025-12-02T10:30:00Z"
}
```

### Metrics Info
```bash
curl http://localhost:8080/api/metrics
```

Response:
```json
{
  "metrics_enabled": true,
  "max_routines": 0,
  "shutdown_timeout": "30s",
  "update_interval": "2s",
  "total_goroutines": 8,
  "total_apps": 2,
  "total_local_managers": 2,
  "metrics_endpoint": "http://localhost:19090/metrics"
}
```

## Prometheus Queries

Once Prometheus is running, you can query metrics:

### Total Goroutines
```promql
goroutine_manager_global_goroutines_total
```

### Goroutines by App
```promql
goroutine_manager_app_goroutines{app_name="api-server"}
goroutine_manager_app_goroutines{app_name="worker-pool"}
```

### Operation Rate
```promql
rate(goroutine_manager_operations_goroutine_operations_total[5m])
```

### Error Rate
```promql
rate(goroutine_manager_operations_errors_total[5m])
```

### P95 Latency
```promql
histogram_quantile(0.95, rate(goroutine_manager_operations_goroutine_operation_duration_seconds_bucket[5m]))
```

## Grafana Dashboard

### Import Dashboard

1. **Download the Dashboard**
   ```bash
   wget https://raw.githubusercontent.com/neerajchowdary889/GoRoutinesManager/main/metrics/Dashboard/grafana-dashboard.json
   ```

2. **Import into Grafana**
   - Open Grafana: http://localhost:3000
   - Login with `admin`/`admin`
   - Go to **Dashboards** → **Import**
   - Upload the JSON file or paste the content
   - Select **Prometheus** as data source
   - Click **Import**

### Dashboard Features

The dashboard includes:
- **Overview**: Total goroutines, apps, local managers
- **Goroutine Metrics**: Counts by app and function
- **Operation Metrics**: Rates and latencies
- **Shutdown Metrics**: Duration and remaining goroutines
- **Configuration**: Current settings

## Configuration

### Metrics Configuration

The example enables metrics with a 2-second update interval:

```go
globalMgr.UpdateMetadata(Global.SET_METRICS_URL, []interface{}{true, "", 2 * time.Second})
```

### Prometheus Configuration

Edit `prometheus.yml` to adjust:
- `scrape_interval`: How often to scrape metrics (default: 10s)
- `targets`: Metrics endpoint address (default: `host.docker.internal:19090`)

For local Prometheus (not Docker), change:
```yaml
targets: ['localhost:19090']
```

### Grafana Configuration

Grafana is pre-configured with:
- Prometheus data source
- Performance optimizations
- Auto-provisioning

## Troubleshooting

### Port Already in Use

If port 19090, 8080, 9090, or 3000 is in use:

1. **Change metrics port** in `main.go`:
   ```go
   metricsPort := ":19091"  // Use different port
   ```

2. **Update Prometheus config** in `prometheus.yml`:
   ```yaml
   targets: ['host.docker.internal:19091']
   ```

3. **Change Docker ports** in `docker-compose.yml`:
   ```yaml
   ports:
     - "9091:9090"  # Prometheus
     - "3001:3000"  # Grafana
   ```

### Prometheus Not Scraping

1. **Check Prometheus targets**: http://localhost:9090/targets
2. **Verify metrics endpoint**: http://localhost:19090/metrics
3. **Check Docker logs**: `docker-compose logs prometheus`

### No Metrics Showing

1. **Verify metrics are enabled**: Check `/api/metrics` endpoint
2. **Check collector is running**: Look for "Metrics collector started" in logs
3. **Wait a few seconds**: Metrics update every 2 seconds

### Docker Issues

**Check if services are running:**
```bash
docker-compose ps
```

**View logs:**
```bash
docker-compose logs prometheus
docker-compose logs grafana
```

**Restart services:**
```bash
docker-compose restart
```

## Code Structure

```
example-metrics/
├── main.go                 # Main service code
├── go.mod                  # Go module dependencies
├── docker-compose.yml      # Docker Compose configuration
├── prometheus.yml          # Prometheus configuration
├── run-with-prometheus.sh  # Helper script
├── grafana/
│   ├── grafana.ini        # Grafana configuration
│   └── provisioning/
│       └── datasources/
│           └── prometheus.yml  # Prometheus data source
└── README.md              # This file
```

## Key Code Patterns

### 1. Initialize Global Manager with Metrics

```go
globalMgr := Global.NewGlobalManager()
globalMgr.Init()
globalMgr.UpdateMetadata(Global.SET_METRICS_URL, []interface{}{true, "", 2 * time.Second})
metrics.StartCollector()
```

### 2. Create Metrics Server

```go
mux := http.NewServeMux()
mux.Handle("/metrics", metrics.GetMetricsHandler())
metricsServer := &http.Server{
    Addr:    ":19090",
    Handler: mux,
}
go metricsServer.ListenAndServe()
```

### 3. Spawn Tracked Goroutines

```go
localMgr.Go("job-processor", func(ctx context.Context) error {
    // Worker logic
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            // Do work
        }
    }
}, Local.AddToWaitGroup("job-processor"))
```

### 4. Graceful Shutdown

```go
metrics.StopCollector()
metricsServer.Shutdown(ctx)
globalMgr.Shutdown(true)
```

## Next Steps

1. **Customize Workers**: Add your own worker functions
2. **Add More Apps**: Create additional app managers
3. **Create Custom Metrics**: Add application-specific metrics
4. **Build Dashboards**: Create Grafana dashboards for your use case
5. **Set Up Alerts**: Configure Prometheus alerts for critical metrics

## Resources

- [Metrics Documentation](../../metrics/README.md)
- [Metrics API Reference](../../metrics/API.md)
- [Grafana Dashboard Guide](../../metrics/Dashboard/GRAFANA_DASHBOARD.md)
- [Main README](../../README.md)

## Support

For issues or questions:
- Check the [main documentation](../../README.md)
- Review the [metrics documentation](../../metrics/README.md)
- Open an issue on GitHub

