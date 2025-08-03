#!/bin/sh
# Health check script for Docker containers

# Check if the service is responding
wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Check if essential endpoints are working
wget --quiet --tries=1 --spider http://localhost:8080/api/v1/check || exit 1

echo "Health check passed"
exit 0
