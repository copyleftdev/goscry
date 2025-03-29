#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== GoScry Docker Test Script ===${NC}"
echo "This script will test the Docker deployment of GoScry"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

# Check if Docker Compose is installed
if ! docker compose version &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    exit 1
fi

# Build the Docker image
echo -e "\n${BLUE}Building Docker image...${NC}"
docker compose build

# Start the Docker container
echo -e "\n${BLUE}Starting GoScry container...${NC}"
docker compose up -d goscry

# Get the API key from the container
echo -e "\n${BLUE}Getting API key...${NC}"
API_KEY=$(docker compose exec goscry sh -c 'echo $GOSCRY_SECURITY_APIKEY')
echo "Using API key: $API_KEY"

# Wait for the container to be ready
echo -e "\n${BLUE}Waiting for GoScry to be ready...${NC}"
attempt=1
max_attempts=10
until docker compose exec goscry wget -q -O- --header="X-API-Key: $API_KEY" http://localhost:8080/health | grep -q "ok" || [ $attempt -eq $max_attempts ]; do
    echo "Attempt $attempt/$max_attempts: GoScry not yet ready, waiting..."
    sleep 5
    ((attempt++))
done

if [ $attempt -eq $max_attempts ]; then
    echo -e "${RED}Failed to connect to GoScry after $max_attempts attempts${NC}"
    docker compose logs goscry
    docker compose down
    exit 1
fi

echo -e "${GREEN}GoScry is up and running!${NC}"

# Test the DOM AST API endpoint
echo -e "\n${BLUE}Testing DOM AST API endpoint...${NC}"

TEST_RESULT=$(docker compose exec goscry wget -q -O- --header="Content-Type: application/json" \
  --header="X-API-Key: $API_KEY" \
  --post-data='{"url":"https://example.com","parent_selector":"body"}' \
  http://localhost:8080/api/v1/dom/ast)

if echo "$TEST_RESULT" | grep -q "nodeType"; then
    echo -e "${GREEN}DOM AST API test successful!${NC}"
    echo "DOM AST response excerpt:"
    echo "$TEST_RESULT" | head -20
else
    echo -e "${RED}DOM AST API test failed${NC}"
    echo "Response:"
    echo "$TEST_RESULT"
fi

# Prompt user if they want to shut down the containers
read -p $'\nDo you want to shut down the containers? (y/n): ' SHUTDOWN
if [[ $SHUTDOWN =~ ^[Yy]$ ]]; then
    echo -e "\n${BLUE}Shutting down containers...${NC}"
    docker compose down
    echo -e "${GREEN}Containers shut down successfully${NC}"
else
    echo -e "\n${GREEN}Containers left running. You can access the API at http://localhost:8090${NC}"
fi

echo -e "\n${BLUE}=== Docker test completed ===${NC}"
