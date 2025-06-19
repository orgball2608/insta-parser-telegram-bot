#!/bin/bash
set -e

# Only set environment variables if they're not already explicitly set
# This allows external database connections to work

# Check if POSTGRES_HOST is set to "localhost" or is empty
if [ "$POSTGRES_HOST" = "localhost" ] || [ -z "$POSTGRES_HOST" ]; then
  echo "Setting PostgreSQL host to 'postgres' container"
  export POSTGRES_HOST=postgres
fi

# Check if REDIS_HOST is set to "localhost" or is empty
if [ "$REDIS_HOST" = "localhost" ] || [ -z "$REDIS_HOST" ]; then
  echo "Setting Redis host to 'redis' container"
  export REDIS_HOST=redis
fi

# Print connection information (for debugging)
echo "PostgreSQL connection: $POSTGRES_HOST:$POSTGRES_PORT"
echo "Redis connection: $REDIS_HOST:$REDIS_PORT"

# Run the main application
exec "$@"
