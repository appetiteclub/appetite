# NATS Installation Guide

This guide covers NATS installation for local development on different platforms.

**For production deployments**, use the Docker image (`nats:2.10-alpine`) via docker-compose or Nomad.

## Quick Start (Docker - Recommended)

If you have Docker installed, no additional setup is needed. The Makefile will automatically use Docker:

```bash
make run-nats
```

To pull the image manually:

```bash
make install-nats-docker
```

---

## Platform-Specific Installation

### Manjaro / Arch Linux

#### Option 1: AUR (Recommended)

```bash
yay -S nats-server
# or
paru -S nats-server
```

#### Option 2: Binary Installation

```bash
# Download latest release
cd /tmp
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-amd64.tar.gz -o nats-server.tar.gz

# Extract and install
tar -xzf nats-server.tar.gz
sudo mv nats-server-v2.10.22-linux-amd64/nats-server /usr/local/bin/
sudo chmod +x /usr/local/bin/nats-server

# Verify installation
nats-server --version
```

#### Option 3: Docker (Fallback)

```bash
sudo pacman -S docker
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
# Log out and back in
```

---

### Debian / Ubuntu

#### Option 1: Package Manager

```bash
# Add NATS repository
curl -fsSL https://packagecloud.io/nats-io/nats-server/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nats-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/nats-archive-keyring.gpg] https://packagecloud.io/nats-io/nats-server/debian/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/nats.list

sudo apt update
sudo apt install nats-server
```

#### Option 2: Binary Installation

```bash
# Download latest release
cd /tmp
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-amd64.tar.gz -o nats-server.tar.gz

# Extract and install
tar -xzf nats-server.tar.gz
sudo mv nats-server-v2.10.22-linux-amd64/nats-server /usr/local/bin/
sudo chmod +x /usr/local/bin/nats-server

# Verify installation
nats-server --version
```

#### Option 3: Docker (Fallback)

```bash
sudo apt update
sudo apt install docker.io
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
# Log out and back in
```

---

### macOS

#### Option 1: Homebrew (Recommended)

```bash
brew install nats-server
```

#### Option 2: Binary Installation

```bash
# Download latest release
cd /tmp
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-darwin-amd64.tar.gz -o nats-server.tar.gz

# For Apple Silicon (M1/M2/M3):
# curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-darwin-arm64.tar.gz -o nats-server.tar.gz

# Extract and install
tar -xzf nats-server.tar.gz
sudo mv nats-server-v2.10.22-darwin-amd64/nats-server /usr/local/bin/
sudo chmod +x /usr/local/bin/nats-server

# Verify installation
nats-server --version
```

#### Option 3: Docker (Fallback)

Install Docker Desktop from https://www.docker.com/products/docker-desktop/

---

### Windows

#### Option 1: Scoop (Recommended)

```powershell
# Install Scoop if not already installed
iwr -useb get.scoop.sh | iex

# Install NATS server
scoop bucket add extras
scoop install nats-server
```

#### Option 2: Binary Installation

1. Download the latest Windows release from:
   https://github.com/nats-io/nats-server/releases

2. Extract `nats-server-v2.10.22-windows-amd64.zip`

3. Move `nats-server.exe` to a directory in your PATH, e.g., `C:\Program Files\NATS\`

4. Add to PATH:
   - Right-click "This PC" → Properties → Advanced System Settings
   - Environment Variables → System Variables → Path → Edit
   - Add `C:\Program Files\NATS\`

5. Verify installation:
   ```powershell
   nats-server --version
   ```

#### Option 3: Docker Desktop (Fallback)

Install Docker Desktop from https://www.docker.com/products/docker-desktop/

---

## Verification

After installation, verify NATS is working:

```bash
# Start NATS
make run-nats

# In another terminal, check if it's running
curl http://localhost:8222/healthz

# Stop NATS
make stop-nats
```

---

## Integration with Appetite

The Appetite project automatically manages NATS:

- **Development (local binaries)**: `make run-all` starts NATS automatically
- **Docker Compose**: NATS container is included in the stack
- **Nomad**: NATS deployment is part of the job definition

### Manual Control

```bash
# Start NATS
make run-nats

# Stop NATS
make stop-nats

# Pull Docker image
make install-nats-docker
```

### Configuration

Default ports:
- Client connections: `4222`
- Monitoring/HTTP: `8222`

Override with environment variables:
```bash
export NATS_PORT=14222
export NATS_MONITOR_PORT=18222
make run-nats
```

---

## Troubleshooting

### Port already in use

```bash
# Check what's using port 4222
lsof -i :4222  # Linux/macOS
netstat -ano | findstr :4222  # Windows

# Stop conflicting process or use different port
export NATS_PORT=14222
make run-nats
```

### Permission denied (Docker)

```bash
# Linux: Add user to docker group
sudo usermod -aG docker $USER
# Log out and back in
```

### NATS won't start

```bash
# Check logs
cat nats.log  # For native installation

# Or check Docker logs
docker logs appetite-nats-dev
```

---

## Additional Resources

- [NATS Official Documentation](https://docs.nats.io/)
- [NATS Server GitHub](https://github.com/nats-io/nats-server)
- [NATS Docker Hub](https://hub.docker.com/_/nats)
