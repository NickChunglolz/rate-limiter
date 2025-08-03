#!/bin/bash
# Network cleanup and setup script

echo "🧹 Cleaning up Docker networks..."

# Stop any running containers that might be using the network
echo "Stopping any running containers..."
docker-compose down 2>/dev/null || true

# Remove existing network if it exists
echo "Removing conflicting networks..."
docker network rm ratelimiter_rate-limiter-network 2>/dev/null || true
docker network rm ratelimiter_default 2>/dev/null || true

# Prune unused networks
echo "Pruning unused networks..."
docker network prune -f

# List current networks to check for conflicts
echo "Current Docker networks:"
docker network ls

echo "✅ Network cleanup complete!"

# Check for IP conflicts
echo "Checking for IP subnet conflicts..."
docker network ls --format "table {{.Name}}\t{{.Driver}}\t{{.Scope}}" | grep bridge

# Show available IP ranges
echo "Checking subnet usage..."
docker network inspect $(docker network ls -q) 2>/dev/null | grep -E '"Subnet"|"Name"' | paste - - | head -10

echo ""
echo "If you still see conflicts, you may need to:"
echo "1. Stop other Docker projects using similar IP ranges"
echo "2. Use 'docker system prune -a' for a complete cleanup"
echo "3. Restart Docker Desktop if on Mac/Windows"
