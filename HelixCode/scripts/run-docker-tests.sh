#!/bin/bash

# HelixCode Docker Testing Script
# This script sets up and runs comprehensive distributed testing using Docker Compose

set -e

echo "🧪 HelixCode Docker Testing Infrastructure"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker and Docker Compose are available
check_dependencies() {
    print_status "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    print_success "Dependencies check passed"
}

# Generate SSH keys for testing
generate_ssh_keys() {
    print_status "Generating SSH keys for worker authentication..."

    mkdir -p test/workers/ssh-keys

    # Remove existing keys if they exist
    rm -f test/workers/ssh-keys/id_rsa test/workers/ssh-keys/id_rsa.pub test/workers/ssh-keys/authorized_keys

    # Generate RSA key pair
    ssh-keygen -t rsa -b 2048 -f test/workers/ssh-keys/id_rsa -N "" -C "helixcode-test"

    # Set correct permissions
    chmod 600 test/workers/ssh-keys/id_rsa
    chmod 644 test/workers/ssh-keys/id_rsa.pub

    # Create authorized_keys for workers
    cp test/workers/ssh-keys/id_rsa.pub test/workers/ssh-keys/authorized_keys

    print_success "SSH keys generated"
}

# Start the testing infrastructure
start_infrastructure() {
    print_status "Starting Docker testing infrastructure..."

    # Use docker compose (newer syntax) if available, fallback to docker-compose
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        COMPOSE_CMD="docker-compose"
    fi

    # Start services
    $COMPOSE_CMD -f docker-compose.test.yml up -d

    print_success "Infrastructure started"

    # Wait for services to be healthy
    print_status "Waiting for services to be ready..."
    sleep 30

    # Check service health
    check_service_health
}

# Check if services are healthy
check_service_health() {
    print_status "Checking service health..."

    # Check PostgreSQL
    if docker exec helixcode-postgres pg_isready -U helix -d helixcode_test &> /dev/null; then
        print_success "PostgreSQL is healthy"
    else
        print_warning "PostgreSQL health check failed"
    fi

    # Check Redis
    if docker exec helixcode-redis redis-cli ping | grep -q PONG; then
        print_success "Redis is healthy"
    else
        print_warning "Redis health check failed"
    fi

    # Check Cognee
    if curl -f http://localhost:8001/health &> /dev/null; then
        print_success "Cognee is healthy"
    else
        print_warning "Cognee health check failed"
    fi

    # Check HelixCode server
    if curl -f http://localhost:8080/health &> /dev/null; then
        print_success "HelixCode server is healthy"
    else
        print_warning "HelixCode server health check failed"
    fi
}

# Run the tests
run_tests() {
    print_status "Running comprehensive test suite..."

    # Run unit tests
    print_status "Running unit tests..."
    docker exec helixcode-test-runner go test -v ./... -coverprofile=/workspace/coverage.out

    # Run integration tests
    print_status "Running integration tests..."
    docker exec helixcode-e2e-test-runner go test -v ./test/e2e/... -tags=e2e

    # Run worker-specific tests
    print_status "Running worker tests..."
    docker exec helixcode-test-runner go test -v ./internal/worker/...

    print_success "Test execution completed"
}

# Generate coverage report
generate_coverage_report() {
    print_status "Generating coverage report..."

    # Copy coverage files from containers
    docker cp helixcode-test-runner:/workspace/coverage.out ./coverage-docker.out 2>/dev/null || true

    # Generate HTML report if coverage file exists
    if [ -f coverage-docker.out ]; then
        go tool cover -html=coverage-docker.out -o coverage-docker.html
        print_success "Coverage report generated: coverage-docker.html"
    fi
}

# Run distributed scenario tests
run_distributed_tests() {
    print_status "Running distributed scenario tests..."

    # Test worker registration
    print_status "Testing worker registration..."
    for i in {1..3}; do
        WORKER_DATA=$(cat <<EOF
{
  "id": "worker-$i",
  "hostname": "worker-$i",
  "capabilities": ["code-generation", "testing"],
  "max_concurrent_tasks": 5,
  "resources": {
    "cpu_count": 2,
    "total_memory": 4294967296,
    "gpu_count": 0
  }
}
EOF
)

        if curl -X POST http://localhost:8080/api/workers/register \
             -H "Content-Type: application/json" \
             -d "$WORKER_DATA" &> /dev/null; then
            print_success "Worker $i registered successfully"
        else
            print_warning "Worker $i registration failed"
        fi
    done

    # Test task creation and distribution
    print_status "Testing task creation and distribution..."
    TASK_DATA=$(cat <<EOF
{
  "type": "code-generation",
  "data": {"description": "Generate REST API endpoints"},
  "priority": 5,
  "criticality": "normal"
}
EOF
)

    if curl -X POST http://localhost:8080/api/tasks \
         -H "Content-Type: application/json" \
         -d "$TASK_DATA" &> /dev/null; then
        print_success "Task created successfully"
    else
        print_warning "Task creation failed"
    fi

    # Test worker task assignment
    print_status "Testing worker task assignment..."
    sleep 5  # Allow time for task processing

    # Check if tasks are being processed
    TASK_COUNT=$(curl -s http://localhost:8080/api/tasks/count 2>/dev/null || echo "0")
    print_status "Active tasks: $TASK_COUNT"
}

# Clean up the testing infrastructure
cleanup() {
    print_status "Cleaning up testing infrastructure..."

    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        COMPOSE_CMD="docker-compose"
    fi

    $COMPOSE_CMD -f docker-compose.test.yml down -v

    # Remove generated SSH keys
    rm -rf test/workers/ssh-keys

    print_success "Cleanup completed"
}

# Show usage
usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  setup     - Set up testing infrastructure"
    echo "  test      - Run all tests"
    echo "  coverage  - Generate coverage report"
    echo "  distributed - Run distributed scenario tests"
    echo "  cleanup   - Clean up testing infrastructure"
    echo "  full      - Run complete test suite (setup -> test -> coverage -> cleanup)"
    echo "  help      - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 full          # Run complete test suite"
    echo "  $0 setup         # Only set up infrastructure"
    echo "  $0 test          # Run tests on existing infrastructure"
}

# Main script logic
case "${1:-full}" in
    "setup")
        check_dependencies
        generate_ssh_keys
        start_infrastructure
        ;;
    "test")
        run_tests
        ;;
    "coverage")
        generate_coverage_report
        ;;
    "distributed")
        run_distributed_tests
        ;;
    "cleanup")
        cleanup
        ;;
    "full")
        check_dependencies
        generate_ssh_keys
        start_infrastructure
        run_tests
        run_distributed_tests
        generate_coverage_report
        cleanup
        print_success "Full test suite completed!"
        ;;
    "help"|*)
        usage
        ;;
esac