#!/bin/bash

# CirrusSync API Setup Script
# This script helps you set up the development environment quickly

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing_deps=()

    if ! command_exists go; then
        missing_deps+=("go (https://golang.org/dl/)")
    fi

    if ! command_exists docker; then
        missing_deps+=("docker (https://docs.docker.com/get-docker/)")
    fi

    if ! command_exists docker-compose; then
        missing_deps+=("docker-compose (https://docs.docker.com/compose/install/)")
    fi

    if ! command_exists openssl; then
        missing_deps+=("openssl")
    fi

    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        exit 1
    fi

    log_success "All prerequisites are installed"
}

# Generate RSA keys for JWT
generate_keys() {
    log_info "Generating RSA keys for JWT..."

    if [ -d "keys" ] && [ -f "keys/private.pem" ] && [ -f "keys/public.pem" ]; then
        log_warning "RSA keys already exist. Skipping generation."
        return
    fi

    mkdir -p keys

    # Generate private key
    openssl genrsa -out keys/private.pem 2048

    # Generate public key
    openssl rsa -in keys/private.pem -pubout -out keys/public.pem

    # Set proper permissions
    chmod 600 keys/private.pem
    chmod 644 keys/public.pem

    log_success "RSA keys generated successfully"
}

# Setup environment file
setup_environment() {
    log_info "Setting up environment configuration..."

    if [ -f ".env" ]; then
        log_warning ".env file already exists. Skipping creation."
        log_info "You can manually edit .env or delete it to regenerate from template."
        return
    fi

    if [ -f ".env.example" ]; then
        cp .env.example .env
        log_success "Environment file created from template"
        log_warning "Please edit .env file with your actual configuration values"
    else
        log_error ".env.example template not found"
        return 1
    fi
}

# Install Go dependencies
install_dependencies() {
    log_info "Installing Go dependencies..."

    go mod download
    go mod verify

    log_success "Dependencies installed successfully"
}

# Setup with Docker
setup_docker() {
    log_info "Setting up with Docker..."

    # Pull all required images
    log_info "Pulling Docker images..."
    docker-compose pull

    # Start services
    log_info "Starting services with Docker Compose..."
    docker-compose up -d postgres redis minio mailhog

    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    sleep 10

    # Check if services are healthy
    local retries=30
    while [ $retries -gt 0 ]; do
        if docker-compose ps | grep -q "Up (healthy)"; then
            break
        fi
        log_info "Waiting for services to be healthy... ($retries retries left)"
        sleep 2
        retries=$((retries - 1))
    done

    if [ $retries -eq 0 ]; then
        log_warning "Services may not be fully ready. You can check with: docker-compose ps"
    fi

    log_success "Docker services are running"
}

# Setup local development (without Docker)
setup_local() {
    log_info "Setting up for local development..."
    log_warning "Make sure you have PostgreSQL, Redis, and S3-compatible storage running locally"
    log_info "Update your .env file with the correct connection details"
}

# Install development tools
install_dev_tools() {
    log_info "Installing development tools..."

    # Install Air for hot reloading
    if ! command_exists air; then
        log_info "Installing Air for hot reloading..."
        go install github.com/cosmtrek/air@latest
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint; then
        log_info "Installing golangci-lint..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi

    log_success "Development tools installed"
}

# Show next steps
show_next_steps() {
    echo
    log_success "Setup completed successfully!"
    echo
    log_info "Next steps:"
    echo "1. Edit the .env file with your actual configuration values"
    echo "2. Start the application:"
    echo "   - With Docker: docker-compose up cirrussync-api"
    echo "   - Local development: air (for hot reload) or go run cmd/main.go"
    echo
    log_info "Useful commands:"
    echo "   - View logs: docker-compose logs -f cirrussync-api"
    echo "   - Stop services: docker-compose down"
    echo "   - Database admin: http://localhost:5050 (pgAdmin)"
    echo "   - Redis admin: http://localhost:8081 (Redis Commander)"
    echo "   - Email testing: http://localhost:8025 (MailHog)"
    echo "   - S3 admin: http://localhost:9001 (MinIO Console)"
    echo
    log_info "API will be available at: http://localhost:8000"
    echo
}

# Main setup function
main() {
    echo "=================================="
    echo "  CirrusSync API Setup Script"
    echo "=================================="
    echo

    # Change to script directory
    cd "$(dirname "$0")/.."

    check_prerequisites
    generate_keys
    setup_environment
    install_dependencies
    install_dev_tools

    # Ask user for setup type
    echo
    log_info "Choose setup type:"
    echo "1. Docker (recommended for development)"
    echo "2. Local (requires manual service setup)"
    read -p "Enter your choice (1 or 2): " choice

    case $choice in
        1)
            setup_docker
            ;;
        2)
            setup_local
            ;;
        *)
            log_warning "Invalid choice. Defaulting to Docker setup."
            setup_docker
            ;;
    esac

    show_next_steps
}

# Handle script interruption
trap 'log_error "Setup interrupted"; exit 1' INT TERM

# Run main function
main "$@"
