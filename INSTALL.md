# Installation Guide

## Quick Install

```bash
# Build and install kubectl plugin
make install-plugin

# Verify installation
kubectl scanfail --help
```

## Troubleshooting Installation

### Issue: `kubectl: unknown command "scanfail"`

**Cause**: The GOPATH/bin directory is not in your PATH.

**Solution**:

1. **Check if plugin is installed:**
   ```bash
   ls -la $(go env GOPATH)/bin/kubectl-scanfail
   ```
   
   If you see the file, it's installed but not in your PATH.

2. **Add GOPATH/bin to PATH (current session):**
   ```bash
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

3. **Make it permanent:**

   **For zsh (macOS default):**
   ```bash
   echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
   source ~/.zshrc
   ```

   **For bash:**
   ```bash
   echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
   source ~/.bashrc
   ```

   **For fish:**
   ```fish
   fish_add_path (go env GOPATH)/bin
   ```

4. **Verify it works:**
   ```bash
   which kubectl-scanfail
   kubectl scanfail --help
   ```

### Issue: `Permission denied`

**Solution**: Make sure the plugin is executable:
```bash
chmod +x $(go env GOPATH)/bin/kubectl-scanfail
```

### Issue: Plugin not found after installation

**Check kubectl plugin discovery:**
```bash
kubectl plugin list
```

This should show `kubectl-scanfail` in the list. If not:

1. Verify GOPATH/bin is in PATH:
   ```bash
   echo $PATH | grep -q "$(go env GOPATH)/bin" && echo "✓ In PATH" || echo "✗ Not in PATH"
   ```

2. Check the plugin exists:
   ```bash
   ls -la $(go env GOPATH)/bin/kubectl-*
   ```

3. Restart your shell:
   ```bash
   exec $SHELL
   ```

## Installation Locations

The `make install-plugin` command installs to:
- **If GOBIN is set**: `$GOBIN/kubectl-scanfail`
- **Otherwise**: `$GOPATH/bin/kubectl-scanfail` (default: `~/go/bin/kubectl-scanfail`)

## Manual Installation

If you prefer to install manually:

```bash
# Build the plugin
go build -o kubectl-scanfail ./cmd/kubectl-scanfail

# Install to a directory in your PATH
sudo mv kubectl-scanfail /usr/local/bin/

# Or install to GOPATH/bin
mv kubectl-scanfail $(go env GOPATH)/bin/

# Make it executable
chmod +x /usr/local/bin/kubectl-scanfail
```

## Alternative: Run Without Installing

You can always run the plugin directly:

```bash
# Build it
make build-plugin

# Run directly
./bin/kubectl-scanfail --all-namespaces
```

## Verification

After installation, verify everything works:

```bash
# Check kubectl can find the plugin
kubectl plugin list | grep scanfail

# Show help
kubectl scanfail --help

# Run a scan
kubectl scanfail --all-namespaces
```

## Uninstallation

To remove the plugin:

```bash
rm $(go env GOPATH)/bin/kubectl-scanfail
```

## Using in CI/CD

For CI/CD pipelines, you may want to install to a specific location:

```bash
# Build
make build-plugin

# Copy to specific location
cp bin/kubectl-scanfail /usr/local/bin/

# Or add bin directory to PATH
export PATH="$PATH:$(pwd)/bin"
kubectl scanfail -A
```

## Platform-Specific Notes

### macOS
- Default shell is zsh, use `~/.zshrc`
- GOPATH defaults to `~/go`
- May need Xcode Command Line Tools: `xcode-select --install`

### Linux
- Usually uses bash, use `~/.bashrc`
- May need to install Go first: `sudo apt-get install golang-go` (Ubuntu/Debian)
- GOPATH defaults to `~/go`

### Windows
- Add `%USERPROFILE%\go\bin` to PATH
- Plugin must be named `kubectl-scanfail.exe`
- Use PowerShell or WSL2 for best experience

## Common PATH Configurations

Add to your shell profile (`.zshrc`, `.bashrc`, etc.):

```bash
# Add Go binary paths to PATH
export GOPATH="$HOME/go"
export GOBIN="$GOPATH/bin"
export PATH="$PATH:$GOBIN"

# Or more simply:
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Getting Help

If you're still having issues:

1. Check Go is installed: `go version`
2. Check kubectl is installed: `kubectl version`
3. Verify PATH: `echo $PATH`
4. Check plugin location: `which kubectl-scanfail`
5. See full kubectl plugin list: `kubectl plugin list`

For more help, see:
- [README.md](README.md) - Full documentation
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [USAGE_NOTES.md](USAGE_NOTES.md) - Usage tips

