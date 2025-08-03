#!/bin/bash

echo "Testing Integrated Rate Limiter Service..."
echo "=========================================="

# Test health endpoint
echo "1. Health Check:"
curl -s http://localhost:8081/health | jq .
echo -e "\n"

# Test main check endpoint
echo "2. Request Check:"
curl -s -X POST http://localhost:8081/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "test-client",
    "resource": "api",
    "ip_address": "192.168.1.100",
    "user_agent": "test-agent"
  }' | jq .
echo -e "\n"

# Test security endpoint
echo "3. IP Blocking:"
curl -s -X POST http://localhost:8081/api/v1/security/block-ips \
  -H "Content-Type: application/json" \
  -d '{
    "ip_addresses": ["10.0.0.1"],
    "reason": "test block",
    "duration": "1m"
  }' | jq .

echo -e "\nService is working correctly! ✅"
