#!/bin/sh
set -e

# Function to wait for a service to be ready
wait_for_service() {
  HOST="$1"
  PORT="$2"
  TIMEOUT="${3:-30}"
  
  echo "Waiting for $HOST:$PORT to be ready..."
  timeout "$TIMEOUT" sh -c "until nc -z $HOST $PORT; do sleep 1; done" || {
    echo "Timed out waiting for $HOST:$PORT"
    exit 1
  }
  echo "$HOST:$PORT is ready!"
}

# Check if we should wait for any services
if [ -n "$WAIT_FOR_SERVICES" ]; then
  for service in $(echo $WAIT_FOR_SERVICES | tr ',' ' '); do
    host=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)
    wait_for_service "$host" "$port"
  done
fi

# Create directories if they don't exist
mkdir -p /var/lib/goscry/data
mkdir -p /var/lib/goscry/logs

# Check if we need to generate a random API key
if [ "$GOSCRY_SECURITY_APIKEY" = "changeme" ] || [ -z "$GOSCRY_SECURITY_APIKEY" ]; then
  if [ "$AUTO_GENERATE_API_KEY" = "true" ]; then
    GOSCRY_SECURITY_APIKEY=$(openssl rand -hex 16)
    echo "Generated new API key: $GOSCRY_SECURITY_APIKEY"
    export GOSCRY_SECURITY_APIKEY
  else
    echo "WARNING: Using default API key. Set GOSCRY_SECURITY_APIKEY environment variable for production."
  fi
fi

# Run the command
exec "$@"
