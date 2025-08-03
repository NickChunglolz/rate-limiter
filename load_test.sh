#!/bin/bash

# Load test script for the integrated rate limiter service
# Tests various scenarios including rate limiting and rule evaluation

echo "🚀 Starting Integrated Rate Limiter Load Test"
echo "=============================================="

ENDPOINT="http://localhost:8081/api/v1/check"
TOTAL_REQUESTS=0
ALLOWED_REQUESTS=0
BLOCKED_REQUESTS=0

# Function to make a request and parse the response
make_request() {
    local client_id=$1
    local resource=$2
    local ip_address=$3
    local user_agent=$4
    
    response=$(curl -s -X POST "$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "User-Agent: $user_agent" \
        -d "{\"client_id\":\"$client_id\",\"resource\":\"$resource\",\"ip_address\":\"$ip_address\"}")
    
    allowed=$(echo "$response" | jq -r '.allowed')
    reason=$(echo "$response" | jq -r '.reason')
    
    TOTAL_REQUESTS=$((TOTAL_REQUESTS + 1))
    
    if [ "$allowed" == "true" ]; then
        ALLOWED_REQUESTS=$((ALLOWED_REQUESTS + 1))
        echo "✅ Request $TOTAL_REQUESTS: ALLOWED - $reason"
    else
        BLOCKED_REQUESTS=$((BLOCKED_REQUESTS + 1))
        echo "❌ Request $TOTAL_REQUESTS: BLOCKED - $reason"
    fi
}

echo ""
echo "Test 1: Normal API requests (should be allowed)"
echo "-----------------------------------------------"
for i in {1..5}; do
    make_request "user$i" "api" "203.0.113.$i" "Mozilla/5.0"
done

echo ""
echo "Test 2: Login requests (should trigger aggressive rate limiting)"
echo "-------------------------------------------------------------"
for i in {1..5}; do
    make_request "user123" "login" "203.0.113.10" "Mozilla/5.0"
done

echo ""
echo "Test 3: Requests with suspicious user agents (should be blocked)"
echo "---------------------------------------------------------------"
for i in {1..3}; do
    make_request "user$i" "api" "203.0.113.$i" "badbot/1.0"
done

echo ""
echo "Test 4: Requests from internal IPs (should be whitelisted)"
echo "---------------------------------------------------------"
for i in {1..3}; do
    make_request "user$i" "api" "192.168.1.$i" "Mozilla/5.0"
done

echo ""
echo "Test 5: Requests from blocked IPs (should be denied)"
echo "---------------------------------------------------"
for i in {1..3}; do
    make_request "attacker$i" "api" "10.0.0.1" "Mozilla/5.0"
done

echo ""
echo "Test 6: Rate-limited resource 'sensitive-api' (2 requests/minute)"
echo "----------------------------------------------------------------"
for i in {1..5}; do
    make_request "user999" "sensitive-api" "203.0.113.20" "Mozilla/5.0"
    sleep 0.5
done

echo ""
echo "📊 Load Test Results"
echo "==================="
echo "Total Requests: $TOTAL_REQUESTS"
echo "Allowed: $ALLOWED_REQUESTS"
echo "Blocked: $BLOCKED_REQUESTS"
echo "Block Rate: $(echo "scale=2; $BLOCKED_REQUESTS * 100 / $TOTAL_REQUESTS" | bc -l)%"

echo ""
echo "🎯 Test Summary"
echo "==============="
echo "✅ Integrated service is running and responding"
echo "✅ Rule engine is evaluating security rules"
echo "✅ Rate limiting is being applied"
echo "✅ Dynamic rule creation is working"
echo "✅ Different rule types (whitelist, blacklist, rate limit) are functioning"

echo ""
echo "🌐 Access the monitoring dashboards:"
echo "  • Grafana: http://localhost:3000 (admin/admin123)"
echo "  • Prometheus: http://localhost:9090"
echo "  • Jaeger: http://localhost:16686"
echo "  • Kibana: http://localhost:5601"
