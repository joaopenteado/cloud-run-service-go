#!/bin/bash
set -e

echo "Building Docker image..."
docker build -t cloud-run-service-go:latest .

echo "Build complete!"
echo ""
echo "To run the container locally:"
echo "  docker run -p 8080:8080 cloud-run-service-go:latest"
echo ""
echo "To deploy to Google Cloud Run:"
echo "  gcloud run deploy cloud-run-service-go \\"
echo "    --image gcr.io/PROJECT_ID/cloud-run-service-go:latest \\"
echo "    --platform managed \\"
echo "    --region us-central1 \\"
echo "    --allow-unauthenticated"
