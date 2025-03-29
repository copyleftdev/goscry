# Docker Deployment Guide for GoScry

This document provides instructions for deploying GoScry using Docker and Docker Compose.

## Prerequisites

- Docker Engine 24.0.0+
- Docker Compose v2.0.0+
- Git (for obtaining the source code)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/copyleftdev/goscry.git
   cd goscry
   ```

2. Create a custom configuration file:
   ```bash
   cp config.example.yaml config.yaml
   ```

3. Update `config.yaml` with your configuration settings:
   - Set `security.apiKey` or set it through environment variable `GOSCRY_SECURITY_APIKEY`
   - Configure additional settings as needed

4. Start the basic stack:
   ```bash
   docker compose up -d
   ```

5. Access the API at http://localhost:8090

## Automated Testing

GoScry includes an automated Docker testing script to validate deployment:

```bash
# Run the automated test script
./docker-test-auto.sh
```

This script:
1. Builds and starts the Docker container
2. Extracts the API key
3. Tests the health endpoint with proper authentication
4. Validates the DOM AST API by fetching example.com
5. Provides useful diagnostics if any step fails

You can also use the interactive test script for manual testing and verification:

```bash
./docker-test.sh
```

## Configuration Options

### Environment Variables

The following environment variables can be set in the `docker-compose.yml` file or in your host environment:

| Variable | Description | Default |
|----------|-------------|---------|
| `GOSCRY_API_KEY` | API key for authenticating requests | `changeme` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `GOSCRY_SERVER_PORT` | Port the server listens on | `8090` |

### Configuration File

Mount a custom `config.yaml` file to override default settings:

```yaml
version: '3.8'
services:
  goscry:
    # ... other settings
    volumes:
      - ./config.yaml:/etc/goscry/config.yaml:ro
```

## Deployment Options

### Basic Deployment

Simple deployment with just the GoScry service:

```bash
docker compose up -d
```

### Production Deployment with Traefik

For production with TLS termination and routing:

1. Update Traefik configuration:
   - Set your domain in `traefik/config/dynamic.yml`
   - Set your email in `traefik/traefik.yml` for Let's Encrypt

2. Start the stack with the production profile:
   ```bash
   docker compose --profile production up -d
   ```

### Monitoring Deployment

Includes Prometheus and Grafana for monitoring:

```bash
docker compose --profile monitoring up -d
```

You can also combine profiles for a complete deployment:
```bash
docker compose --profile production --profile monitoring up -d
```

Access Grafana at http://localhost:3000 (default credentials: admin/admin)

## Resource Requirements

- Minimum: 1 CPU, 1GB RAM
- Recommended: 2 CPUs, 2GB RAM
- Production: 4 CPUs, 4GB RAM (depending on expected load)

## Docker Image Customization

The Docker image can be customized by modifying the Dockerfile. Key aspects:

1. **Base Image**: Uses `zenika/alpine-chrome` which provides Chrome/Chromium necessary for the browser automation
2. **Multi-stage build**: Optimizes image size by building in one container and copying only necessary files to the runtime container
3. **Non-root user**: Runs as a non-privileged user for better security
4. **Volumes**: Persists data through a Docker volume at `/var/lib/goscry`

## Performance Tuning

For high-throughput environments:

1. Increase `GOSCRY_BROWSER_MAXSESSIONS` (default: 10) to handle more concurrent browser sessions
2. Adjust resource limits in `docker-compose.yml`
3. Consider using a separate database service for task storage (requires code modifications)

## Troubleshooting

### Common Issues

1. **Container fails to start**: 
   - Check logs: `docker compose logs goscry`
   - Ensure config.yaml is properly formatted

2. **Browser automation issues**:
   - Ensure the container has enough resources (memory and CPU)
   - Check browser logs: `docker compose exec goscry cat /var/log/chrome_debug.log`

3. **Network connectivity issues**:
   - Ensure required ports are not blocked by firewall
   - Check Traefik logs for routing issues: `docker compose logs traefik`

## Security Considerations

1. **API Key**: Always change the default API key in production
2. **TLS**: Use the Traefik setup with Let's Encrypt for TLS termination
3. **Network Isolation**: The docker-compose setup uses a dedicated network for service communication
4. **Regular Updates**: Keep the Docker images updated for security patches
