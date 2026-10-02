# K8s App Plugin - Kubernetes Failure Scanner

A comprehensive Kubernetes scanner that detects failing resources across your cluster. Available as a kubectl plugin, standalone CLI, and desktop GUI application.

## Features

- 🔍 **Comprehensive Scanning**: Detects failures in Pods, Deployments, StatefulSets, DaemonSets, Jobs, CronJobs, Nodes, and PVCs
- 🎯 **Three Deployment Options**: Use as kubectl plugin, standalone CLI, or GUI application
- 🚀 **Shared Core**: All three applications use the same battle-tested scanner logic
- 📊 **Multiple Output Formats**: Table, simple, or JSON output (CLI only)
- ⚡ **Fast and Efficient**: Concurrent scanning with minimal overhead
- 🔄 **Auto-refresh**: GUI supports automatic periodic scanning

## What Counts as "Failing"?

The scanner uses intelligent heuristics to detect various failure conditions:

- **Pods**: Container waiting states (CrashLoopBackOff, ImagePullBackOff), pods not ready, or failed phase
- **Deployments/StatefulSets**: Available replicas less than desired replicas
- **DaemonSets**: Any unavailable pods
- **Jobs**: Failed jobs with no successes
- **CronJobs**: Most recent job failed
- **Nodes**: Ready condition is False or Unknown
- **PVCs**: Phase is Pending or Lost

## Installation

### Prerequisites

- Go 1.21 or later
- kubectl (for plugin installation)
- Docker (for KIND cluster testing)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/0xNiel/k8s-app-plugin.git
cd k8s-app-plugin

# Download dependencies
make deps

# Build all binaries
make build

# Install kubectl plugin
make install-plugin
```

## Usage

### 1. kubectl Plugin

Once installed, use it as a native kubectl command:

```bash
# Scan all namespaces
kubectl scanfail --all-namespaces

# Scan specific namespace
kubectl scanfail --namespace default

# Use custom kubeconfig
kubectl scanfail --kubeconfig ~/.kube/custom-config
```

### 2. Standalone CLI (kscan)

More features and output options:

```bash
# Basic usage
./bin/kscan --all-namespaces

# With different output formats
./bin/kscan -A -o json
./bin/kscan -n default -o simple
./bin/kscan -o table  # default

# Show version
./bin/kscan --version
```

### 3. GUI Application (KScan GUI)

Run the desktop application:

```bash
./bin/kscan-gui
```

Features:
- Visual interface for cluster scanning
- Auto-refresh with configurable intervals (10s, 30s, 1m, 5m)
- Easy configuration of kubeconfig and namespace
- Real-time results display

## Development

### Building

```bash
# Build all binaries
make build

# Build specific binary
make build-plugin   # kubectl plugin
make build-cli      # standalone CLI
make build-gui      # GUI application
```

### Testing

```bash
# Run unit tests
make test

# Run tests with verbose output
make test-verbose

# Generate coverage report
make test-coverage

# Run linters
make lint

# Format code
make fmt
```

### KIND Cluster Testing

Create a local test cluster and deploy example resources:

```bash
# Create KIND cluster
make cluster-up

# Deploy example resources (healthy and failing)
make deploy-examples

# Run scanner against cluster
make scan

# Clean up
make undeploy-examples
make cluster-down
```

### Full Demo

Run a complete demonstration:

```bash
# Creates cluster, deploys examples, and runs scanner
make demo

# Or run full integration test
make full-test
```

## Makefile Targets

### Build Targets
- `make build` - Build all binaries
- `make build-plugin` - Build kubectl plugin
- `make build-cli` - Build standalone CLI
- `make build-gui` - Build GUI application
- `make install-plugin` - Install kubectl plugin to GOBIN

### Development Targets
- `make run-cli` - Run standalone CLI
- `make run-gui` - Run GUI application
- `make test` - Run tests
- `make test-verbose` - Run tests with verbose output
- `make lint` - Run linters
- `make fmt` - Format code

### KIND Cluster Targets
- `make cluster-up` - Create KIND cluster
- `make cluster-down` - Delete KIND cluster
- `make cluster-restart` - Restart KIND cluster
- `make cluster-info` - Show cluster info

### Example Resource Targets
- `make deploy-examples` - Deploy all example resources
- `make deploy-healthy` - Deploy healthy resources only
- `make deploy-failing` - Deploy failing resources only
- `make undeploy-examples` - Remove all example resources

### Utility Targets
- `make clean` - Clean build artifacts
- `make deps` - Download dependencies
- `make tools` - Install development tools
- `make scan` - Run scanner against current cluster
- `make help` - Show all available targets

## Project Structure

```
.
├── cmd/
│   ├── kubectl-scanfail/   # kubectl plugin
│   ├── kscan/              # Standalone CLI
│   └── kscan-gui/          # GUI application
├── pkg/
│   └── scanner/            # Shared scanner logic
│       ├── scanner.go      # Core scanner implementation
│       └── scanner_test.go # Comprehensive tests
├── examples/
│   ├── healthy-resources.yaml   # Healthy resource examples
│   └── failing-resources.yaml   # Failing resource examples
├── Makefile                # Build and development tasks
├── go.mod                  # Go module definition
└── README.md               # This file
```

## Architecture

The project follows a modular architecture:

1. **Shared Scanner Package** (`pkg/scanner`): Contains all the core logic for detecting failures
2. **kubectl Plugin** (`cmd/kubectl-scanfail`): Minimal wrapper around the scanner for kubectl integration
3. **Standalone CLI** (`cmd/kscan`): Enhanced CLI with multiple output formats
4. **GUI Application** (`cmd/kscan-gui`): Desktop app built with Fyne framework

All three applications import and use the same scanner package, ensuring consistency and maintainability.

## Testing Strategy

The project includes:

- **Unit Tests**: Comprehensive tests for all scanner functions using fake Kubernetes clients
- **Integration Tests**: Example resources that can be deployed to a real cluster
- **CI/CD Ready**: Make targets for continuous integration (`make ci`)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters: `make pre-commit`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Troubleshooting

### Plugin not found after installation

Make sure `$GOBIN` is in your `$PATH`:

```bash
export PATH=$PATH:$(go env GOBIN)
```

### GUI not starting

Install required dependencies for Fyne (varies by OS):

- **macOS**: No additional dependencies
- **Linux**: `sudo apt-get install libgl1-mesa-dev xorg-dev`
- **Windows**: No additional dependencies

### Scanner not detecting failures

Wait a moment after deploying failing resources - some failures take time to manifest (e.g., ImagePullBackOff has a backoff period).

## Future Enhancements

- [ ] Support for custom resource definitions (CRDs)
- [ ] Configurable failure criteria via config file
- [ ] Export results to file (CSV, JSON, YAML)
- [ ] Webhook notifications for detected failures
- [ ] Historical trending of failures
- [ ] Integration with monitoring systems (Prometheus, Grafana)

## Credits

Built with:
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes client library
- [Fyne](https://fyne.io/) - GUI framework
- [KIND](https://kind.sigs.k8s.io/) - Kubernetes in Docker for testing

