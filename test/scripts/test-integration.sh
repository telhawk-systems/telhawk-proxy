#!/bin/bash

# GoTrack JS ↔ Go Integration Test Script
# This script starts the Go server and provides testing utilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 GoTrack Integration Test Setup${NC}"

# Check if Go binary exists
if [[ ! -f "./gotrack" ]]; then
    echo -e "${YELLOW}⚠️  Building Go application...${NC}"
    go build -o ./gotrack ./cmd/gotrack
fi

# Function to test endpoints
test_endpoints() {
    echo -e "\n${BLUE}🧪 Testing Integration...${NC}"
    
    # Test health endpoint
    echo -e "${YELLOW}Testing health endpoint...${NC}"
    if curl -s http://localhost:19890/healthz > /dev/null; then
        echo -e "${GREEN}✅ Health endpoint working${NC}"
    else
        echo -e "${RED}❌ Health endpoint failed${NC}"
        return 1
    fi
    
    # Test /collect endpoint with proper payload
    echo -e "${YELLOW}Testing /collect endpoint...${NC}"
    RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{
            "event_id": "test_integration_123",
            "ts": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
            "type": "pageview",
            "device": {
                "ua": "GoTrack-Integration-Test/1.0",
                "language": "en-US",
                "viewport_w": 1920,
                "viewport_h": 1080
            },
            "session": {
                "session_id": "integration_test_session"
            },
            "url": {
                "referrer": "https://github.com/gotrack/integration-test"
            }
        }' \
        http://localhost:19890/collect)
    
    if echo "$RESPONSE" | grep -q '"status":"ok"'; then
        echo -e "${GREEN}✅ /collect endpoint working${NC}"
        echo "   Response: $RESPONSE"
    else
        echo -e "${RED}❌ /collect endpoint failed${NC}"
        echo "   Response: $RESPONSE"
        return 1
    fi
    
    # Test /px.gif endpoint
    echo -e "${YELLOW}Testing /px.gif endpoint...${NC}"
    if curl -s "http://localhost:19890/px.gif?e=test&url=https://example.com/integration-test" | xxd | head -n 1 | grep -q "GIF89a"; then
        echo -e "${GREEN}✅ /px.gif endpoint working${NC}"
    else
        echo -e "${RED}❌ /px.gif endpoint failed${NC}"
        return 1
    fi
    
    # Check event logs
    echo -e "${YELLOW}Checking event logs...${NC}"
    if [[ -f "./integration-test.ndjson" ]]; then
        EVENTS=$(wc -l < ./integration-test.ndjson)
        echo -e "${GREEN}✅ Found $EVENTS events in log${NC}"
        echo -e "${YELLOW}Latest event:${NC}"
        tail -n 1 ./integration-test.ndjson | jq '.' 2>/dev/null || tail -n 1 ./integration-test.ndjson
    else
        echo -e "${YELLOW}⚠️  No event log file found${NC}"
    fi
    
    return 0
}

# Function to show test page instructions
show_test_page_info() {
    echo -e "\n${BLUE}🌐 Browser Testing${NC}"
    echo -e "${YELLOW}Test page available at:${NC}"
    echo -e "  ${GREEN}file://$(pwd)/test-pixel.html${NC}"
    echo -e "\n${YELLOW}Or serve with Python:${NC}"
    echo -e "  ${GREEN}python3 -m http.server 8080${NC}"
    echo -e "  ${GREEN}Then visit: http://localhost:8080/test-pixel.html${NC}"
}

# Function to start server
start_server() {
    echo -e "\n${BLUE}🏁 Starting GoTrack Server${NC}"
    echo -e "${YELLOW}Configuration:${NC}"
    echo -e "  Server: http://localhost:19890"
    echo -e "  Outputs: log"
    echo -e "  Log file: ./integration-test.ndjson"
    echo -e "  CORS: enabled""
    
    echo -e "\n${YELLOW}Press Ctrl+C to stop the server${NC}\n"
    
    # Start the server
    OUTPUTS=log \
    LOG_PATH=./integration-test.ndjson \
    SERVER_ADDR=":19890" \
    ./gotrack
}

# Main script logic
case "${1:-}" in
    "test")
        echo -e "${BLUE}Running integration tests only...${NC}"
        test_endpoints
        ;;
    "info")
        show_test_page_info
        ;;
    "clean")
        echo -e "${YELLOW}Cleaning up test files...${NC}"
        rm -f ./integration-test.ndjson
        echo -e "${GREEN}✅ Cleanup complete${NC}"
        ;;
    *)
        echo -e "${YELLOW}Starting server and running tests...${NC}"
        
        # Start server in background for testing
        OUTPUTS=log \
        LOG_PATH=./integration-test.ndjson \
        SERVER_ADDR=":19890" \
        ./gotrack &
        
        SERVER_PID=$!
        echo -e "${YELLOW}Server started with PID: $SERVER_PID${NC}"
        
        # Wait for server to start
        sleep 2
        
        # Run tests
        if test_endpoints; then
            echo -e "\n${GREEN}🎉 All integration tests passed!${NC}"
            show_test_page_info
        else
            echo -e "\n${RED}❌ Integration tests failed${NC}"
            kill $SERVER_PID 2>/dev/null || true
            exit 1
        fi
        
        # Kill background server and start in foreground
        kill $SERVER_PID 2>/dev/null || true
        sleep 1
        
        # Start server in foreground
        start_server
        ;;
esac