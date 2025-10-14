# Usage Notes

## Exit Codes and Make Targets

### Understanding Exit Codes

The scanner binaries are designed with CI/CD in mind:

- **Exit Code 0**: No failures detected ✅
- **Exit Code 1**: Failures detected ⚠️

This is intentional and allows you to use the scanner in automated pipelines where you want the build to fail if problems are detected.

### Makefile Scan Targets

#### `make scan` - Interactive Mode (Recommended for CLI use)

Use this for interactive scanning. It always exits successfully even when failures are found:

```bash
make scan
# Output: Failures listed, then "INFO: Scan complete - check results above"
# Exit code: 0 (even if failures found)
```

**Best for:**
- Interactive use
- Quick cluster checks
- Development workflow

#### `make scan-strict` - CI/CD Mode

Use this in CI/CD pipelines where you want the build to fail if issues are detected:

```bash
make scan-strict
# Output: Failures listed
# Exit code: 1 if failures found, 0 if no failures
```

**Best for:**
- CI/CD pipelines
- Automated testing
- Pre-deployment validation
- Gating deployments based on cluster health

### Direct Binary Usage

You can also run the binaries directly for more control:

```bash
# Exits with code 1 if failures found (default behavior)
./bin/kscan --all-namespaces

# Ignore exit code in shell
./bin/kscan -A || true

# Check exit code programmatically
./bin/kscan -n default
if [ $? -eq 0 ]; then
  echo "No failures!"
else
  echo "Failures detected"
fi
```

## CI/CD Integration Examples

### GitHub Actions

```yaml
- name: Check for cluster failures
  run: |
    make scan-strict
  # Job fails if failures are detected
```

### GitLab CI

```yaml
check_cluster:
  script:
    - make scan-strict
  # Pipeline fails if failures are detected
```

### Jenkins

```groovy
stage('Scan Cluster') {
  steps {
    sh 'make scan-strict'
    // Build fails if failures are detected
  }
}
```

### Pre-Deployment Gate

```bash
#!/bin/bash
# deploy.sh

echo "Checking cluster health before deployment..."
if ! make scan-strict; then
  echo "❌ Deployment blocked: Cluster has failing resources"
  echo "Fix the issues above before deploying"
  exit 1
fi

echo "✅ Cluster healthy, proceeding with deployment"
# ... your deployment commands
```

## Output Formats for Different Use Cases

### Human-Readable (Default)

```bash
make scan
# or
./bin/kscan -A
```

### JSON for Parsing

```bash
./bin/kscan -A -o json > failures.json
cat failures.json | jq '.[] | select(.kind=="Pod")'
```

### Simple Format for Logs

```bash
./bin/kscan -A -o simple >> cluster-health.log
```

## Common Workflows

### Daily Health Check

```bash
# Add to crontab
0 9 * * * cd /path/to/k8s-app-plugin && make scan | mail -s "Daily Cluster Health" ops@example.com
```

### Pre-Commit Hook

```bash
# .git/hooks/pre-push
#!/bin/bash
echo "Checking cluster health before push..."
make scan-strict || {
  echo "Warning: Cluster has failures. Push anyway? (y/n)"
  read answer
  if [ "$answer" != "y" ]; then
    exit 1
  fi
}
```

### Multi-Cluster Monitoring

```bash
#!/bin/bash
# check-all-clusters.sh

for context in prod staging dev; do
  echo "=== Checking $context ==="
  kubectl config use-context $context
  ./bin/kscan -A -o simple
  echo ""
done
```

## Troubleshooting

### "make: *** [scan] Error 1" but scanner shows results

**This is the expected behavior!** The scanner found failures, which is what you want to know.

- Use `make scan` for interactive use (no error)
- Use `make scan-strict` for CI/CD (error on failures)

### No failures shown but resources are failing

Wait a bit - some failures take time to manifest:
- ImagePullBackOff has exponential backoff
- CrashLoopBackOff has backoff periods
- Jobs may be retrying

```bash
# Deploy examples and wait
make deploy-failing
sleep 60  # Wait for failures to manifest
make scan
```

### Want to customize exit behavior

Edit the binaries directly or use shell logic:

```bash
# Force success
./bin/kscan -A || true

# Force failure even on success
./bin/kscan -A && false

# Custom logic
./bin/kscan -A -o json > out.json
if [ $(cat out.json | jq 'length') -gt 5 ]; then
  echo "Too many failures!"
  exit 1
fi
```

## Quick Reference

| Command | Exit on Failure | Output Format | Best For |
|---------|----------------|---------------|----------|
| `make scan` | No ❌ | Table | Interactive use |
| `make scan-strict` | Yes ✅ | Table | CI/CD pipelines |
| `./bin/kscan -A` | Yes ✅ | Table | Direct scanning |
| `./bin/kscan -A -o json` | Yes ✅ | JSON | Parsing/automation |
| `./bin/kscan -A -o simple` | Yes ✅ | Simple | Logs |
| `kubectl scanfail -A` | Yes ✅ | Table | Native kubectl |
| `./bin/kscan-gui` | N/A | GUI | Continuous monitoring |

## Getting Help

```bash
# Show all make targets
make help

# CLI help
./bin/kscan --help

# kubectl plugin help
kubectl scanfail --help

# Check version
./bin/kscan --version
```

