# Quick Start Guide

Get up and running with the Kubernetes Failure Scanner in under 5 minutes!

## Prerequisites

- Go 1.21+
- Docker (for KIND cluster testing)
- kubectl installed

## Installation

```bash
# Clone and build
git clone https://github.com/0xNiel/k8s-app-plugin.git
cd k8s-app-plugin
make deps
make build
```

## Quick Test with KIND

The easiest way to try the scanner is with a local KIND cluster:

```bash
# 1. Create a KIND cluster with example resources
make cluster-up
make deploy-examples

# 2. Wait for resources to start failing (about 30 seconds)
sleep 30

# 3. Run the scanner
make scan
```

Expected output:
```
Found X failing resource(s):

KIND                     NAMESPACE   NAME                    REASON               DETAILS
----                     ---------   ----                    ------               -------
DaemonSet                default     failing-daemonset       PodsUnavailable      ...
Deployment               default     failing-deployment      ReplicasUnavailable  ...
Job                      default     failing-job             JobFailed            ...
PersistentVolumeClaim    default     failing-pvc             PVCPending           ...
Pod                      default     failing-pod-crash       ContainerWaiting     ...
Pod                      default     failing-pod-image       ContainerWaiting     ...
StatefulSet              default     failing-statefulset     ReplicasUnavailable  ...
```

## Usage Examples

### Standalone CLI (kscan)

```bash
# Scan all namespaces (table format)
./bin/kscan --all-namespaces

# Scan specific namespace
./bin/kscan --namespace kube-system

# JSON output
./bin/kscan -A -o json

# Simple output
./bin/kscan -n default -o simple

# Check version
./bin/kscan --version
```

### kubectl Plugin

```bash
# Install the plugin
make install-plugin

# Use as kubectl command
kubectl scanfail --all-namespaces
kubectl scanfail -n default
kubectl scanfail --kubeconfig ~/.kube/prod-config
```

### GUI Application

```bash
# Run the GUI
./bin/kscan-gui
```

Then:
1. Enter your kubeconfig path (default: `~/.kube/config`)
2. Enter namespace (leave empty for all namespaces)
3. Click "Scan Cluster"
4. Optionally enable "Auto-refresh" with desired interval

## Understanding the Output

### Failure Reasons

- **ContainerWaiting**: Pod container stuck waiting (CrashLoopBackOff, ImagePullBackOff, etc.)
- **PodNotReady**: Pod is running but not in Ready state
- **PhaseFailed**: Pod phase is Failed
- **ReplicasUnavailable**: Deployment/StatefulSet has fewer available replicas than desired
- **PodsUnavailable**: DaemonSet has unavailable pods
- **JobFailed**: Job has failed attempts with no successes
- **LastJobFailed**: CronJob's most recent job failed
- **NodeNotReady**: Node Ready condition is False or Unknown
- **PVCPending**: PersistentVolumeClaim is in Pending state
- **PVCLost**: PersistentVolumeClaim is in Lost state

## Common Scenarios

### 1. Monitor Production Cluster

```bash
# Install plugin
make install-plugin

# Create an alias for your production cluster
alias prod-scan='kubectl scanfail --kubeconfig ~/.kube/prod-config -A'

# Run scan
prod-scan
```

### 2. CI/CD Integration

```bash
# Exit with error code if failures found
./bin/kscan -A -o json > failures.json
if [ $? -ne 0 ]; then
  echo "Failures detected in cluster!"
  cat failures.json
  exit 1
fi
```

### 3. Multiple Clusters

```bash
# Scan multiple clusters
for ctx in $(kubectl config get-contexts -o name); do
  echo "Scanning context: $ctx"
  kubectl --context $ctx scanfail -A
  echo "---"
done
```

### 4. Scheduled Monitoring (GUI)

1. Run `./bin/kscan-gui`
2. Configure kubeconfig and namespace
3. Enable "Auto-refresh"
4. Select interval (e.g., "1m")
5. Leave running in background

## Cleanup

```bash
# Remove example resources
make undeploy-examples

# Delete KIND cluster
make cluster-down

# Clean build artifacts
make clean
```

## Development

```bash
# Run tests
make test

# Run linters
make lint

# Format code
make fmt

# Run all checks (for CI)
make ci
```

## Troubleshooting

### "Error creating scanner: ..."

Check that your kubeconfig is valid and you have access to the cluster:
```bash
kubectl cluster-info
```

### No failures detected but resources are failing

Wait a bit longer - some failures (like ImagePullBackOff) take time to manifest due to backoff periods.

### Plugin not found after install

Add GOBIN to your PATH:
```bash
export PATH=$PATH:$(go env GOBIN)
# Add to ~/.bashrc or ~/.zshrc to persist
```

### GUI won't start (Linux)

Install required dependencies:
```bash
sudo apt-get install libgl1-mesa-dev xorg-dev
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check available [Makefile targets](Makefile) with `make help`
- Customize failure detection criteria in `pkg/scanner/scanner.go`
- Deploy to production clusters

## Support

For issues or questions:
- Open an issue on GitHub
- Check the [README.md](README.md) for detailed documentation
- Review the [examples/](examples/) directory for resource examples

Happy scanning! 🔍

