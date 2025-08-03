#!/bin/bash
# Load testing script for rate limiter

TARGET_URL=${TARGET_URL:-"http://integrated-service:8080"}
TEST_DURATION=${TEST_DURATION:-"60s"}
CONCURRENT_USERS=${CONCURRENT_USERS:-"10"}

echo "Starting load test..."
echo "Target URL: $TARGET_URL"
echo "Duration: $TEST_DURATION"
echo "Concurrent Users: $CONCURRENT_USERS"

# Function to make a request
make_request() {
    local user_id=$1
    local resource=$2
    
    curl -s -X POST "$TARGET_URL/api/v1/check" \
        -H "Content-Type: application/json" \
        -d "{
            \"client_id\": \"user_$user_id\",
            \"resource\": \"$resource\",
            \"ip_address\": \"192.168.1.$user_id\",
            \"user_agent\": \"LoadTest/1.0\"
        }" \
        -w "Status: %{http_code}, Time: %{time_total}s\n" \
        -o /dev/null
}

# Run load test
echo "Starting concurrent requests..."

# Test different resources
resources=("api" "login" "upload")

# Background processes for concurrent users
for i in $(seq 1 $CONCURRENT_USERS); do
    (
        end_time=$((SECONDS + ${TEST_DURATION%s}))
        request_count=0
        
        while [ $SECONDS -lt $end_time ]; do
            resource=${resources[$((RANDOM % ${#resources[@]}))]}
            make_request $i $resource
            request_count=$((request_count + 1))
            
            # Random delay between requests (0.1 to 1 second)
            sleep $(echo "scale=1; $RANDOM/32767" | bc -l)
        done
        
        echo "User $i completed $request_count requests"
    ) &
done

# Wait for all background processes
wait

echo "Load test completed!"

# Get final statistics
echo "Getting final statistics..."
curl -s "$TARGET_URL/api/v1/ratelimit/stats?client_id=user_1" | jq . || echo "Statistics not available"
