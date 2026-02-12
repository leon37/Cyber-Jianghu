#!/bin/bash

# Cyber-Jianghu Quick Start Script
# This script helps you quickly start all required services

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_error ".env file not found! Please copy .env.example to .env and configure it first."
    print_info "Run: cp .env.example .env"
    print_info "Then edit .env with your API keys and configuration"
    exit 1
fi

# Load environment variables
source .env

# Check for required variables
if [ -z "$ZHIPUAI_API_KEY" ]; then
    print_error "ZHIPUAI_API_KEY is not set in .env"
    exit 1
fi

print_info "Starting Cyber-Jianghu services..."

# Start Docker services
print_info "Starting Docker Compose services..."
docker-compose up -d

# Wait for services to be ready
print_info "Waiting for services to start..."

# Check ComfyUI
if [ "$COMFYUI_URL" = "http://comfyui:8188" ]; then
    print_info "Waiting for ComfyUI to be ready..."

    # Try to connect to ComfyUI
    for i in {1..30}; do
        if curl -s http://localhost:8188/system_stats > /dev/null 2>&1; then
            print_info "ComfyUI is ready!"
            break
        fi
        if [ $((i % 5)) -eq 0 ]; then
            echo -n "."
        fi
        sleep 1
    done

    if [ $i -eq 30 ]; then
        print_warn "ComfyUI not responding after 30 seconds"
        print_warn "You may need to start it manually or check the logs"
    fi
fi

# Check server health
print_info "Checking server health..."
for i in {1..10}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_info "Server is ready!"
        break
    fi
    sleep 1
done

if [ $i -eq 10 ]; then
    print_error "Server failed to start within 10 seconds"
    print_info "Check logs: docker-compose logs server"
fi

# Print service URLs
echo ""
print_info "============================================"
print_info "Service URLs"
print_info "============================================"
echo ""
print_info "Cyber-Jianghu Server: http://localhost:8080"
print_info "ComfyUI:          http://localhost:8188"
print_info "Qdrant:          http://localhost:6333 (or http://localhost:6334/dashboard)"
print_info ""
print_info "To stop services:"
print_info "  docker-compose down"
print_info ""
print_info "To view logs:"
print_info "  docker-compose logs -f"
print_info "============================================"
