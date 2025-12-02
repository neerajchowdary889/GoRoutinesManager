#!/bin/bash

# Script to run example-metrics with Prometheus and Grafana in Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting Prometheus and Grafana in Docker..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 5

# Check if Prometheus is running
if ! curl -s http://localhost:9090/-/healthy > /dev/null; then
    echo "❌ Prometheus is not responding. Check logs with: docker-compose logs"
    docker-compose down
    exit 1
fi

# Check if Grafana is running
if ! curl -s http://localhost:3000/api/health > /dev/null; then
    echo "⚠️  Grafana is not responding yet, but continuing..."
else
    echo "✅ Grafana is running at http://localhost:3000"
fi

echo "✅ Prometheus is running at http://localhost:9090"
echo ""
echo "📊 Service URLs:"
echo "   - Prometheus: http://localhost:9090"
echo "   - Grafana: http://localhost:3000 (admin/admin)"
echo "   - Metrics endpoint: http://localhost:19090/metrics"
echo ""
echo "🧪 Starting example service..."
echo ""

# Run the example service
go run main.go

SERVICE_EXIT_CODE=$?

echo ""
if [ $SERVICE_EXIT_CODE -eq 0 ]; then
    echo "✅ Service stopped gracefully"
    echo ""
    echo "📊 Prometheus and Grafana are still running:"
    echo "   - Prometheus: http://localhost:9090"
    echo "   - Grafana: http://localhost:3000"
    echo ""
    read -p "Press Enter to stop Prometheus and Grafana, or Ctrl+C to keep them running..."
else
    echo "❌ Service exited with code $SERVICE_EXIT_CODE"
fi

echo "🛑 Stopping Prometheus and Grafana..."
docker-compose down

exit $SERVICE_EXIT_CODE

