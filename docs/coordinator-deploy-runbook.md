# DGInf Deploy Runbook

How to build, deploy, and update all DGInf components: the coordinator, provider CLI, and macOS app bundle.

## Infrastructure

| Item | Value |
|------|-------|
| Instance | `dginf-mdm` (`i-01a3a5368995a99aa`) |
| Type | `t3.small` (us-east-1a) |
| Public IP | `34.197.17.112` (Elastic IP) |
| Domain | `inference-test.openinnovation.dev` |
| SSH Key | `~/.ssh/dginf-infra` |
| SSH User | `ubuntu` |
| AWS Profile | `admin` |
| Binary Path | `/usr/local/bin/dginf-coordinator` |
| Service | `dginf-coordinator.service` (systemd) |
| Listens on | `:8080` (proxied by nginx on 443) |

## Environment Variables (set in systemd unit)

- `DGINF_PORT=8080`
- `DGINF_ADMIN_KEY=dginf-admin-key-2026`
- `DGINF_MDM_URL=https://inference-test.openinnovation.dev`
- `DGINF_MDM_API_KEY=dginf-micromdm-api`

## Deploy Steps

### 1. Build the Linux binary

From the repo root:

```bash
cd coordinator
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dginf-coordinator-linux ./cmd/coordinator
```

This produces a statically linked amd64 binary.

### 2. Run tests before deploying

```bash
cd coordinator
go test ./...
```

All tests must pass before deploying.

### 3. Copy the binary to the server

```bash
scp -i ~/.ssh/dginf-infra dginf-coordinator-linux ubuntu@34.197.17.112:/tmp/dginf-coordinator
```

### 4. SSH in and swap the binary

```bash
ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112
```

On the server:

```bash
# Stop the service
sudo systemctl stop dginf-coordinator

# Replace the binary
sudo mv /tmp/dginf-coordinator /usr/local/bin/dginf-coordinator
sudo chmod +x /usr/local/bin/dginf-coordinator

# Start the service
sudo systemctl start dginf-coordinator

# Verify it's running
sudo systemctl status dginf-coordinator
sudo journalctl -u dginf-coordinator -n 20 --no-pager
```

### 5. Verify the deployment

```bash
# Health check
curl https://inference-test.openinnovation.dev/health

# Models endpoint
curl https://inference-test.openinnovation.dev/v1/models
```

## Quick one-liner deploy

From the repo root (builds, copies, restarts in one shot):

```bash
cd coordinator && \
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dginf-coordinator-linux ./cmd/coordinator && \
scp -i ~/.ssh/dginf-infra dginf-coordinator-linux ubuntu@34.197.17.112:/tmp/dginf-coordinator && \
ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112 \
  'sudo systemctl stop dginf-coordinator && \
   sudo mv /tmp/dginf-coordinator /usr/local/bin/dginf-coordinator && \
   sudo chmod +x /usr/local/bin/dginf-coordinator && \
   sudo systemctl start dginf-coordinator && \
   sleep 2 && \
   sudo systemctl status dginf-coordinator --no-pager'
```

## Rollback

If the new binary fails to start:

```bash
# The previous binary isn't kept automatically. If you need rollback,
# keep a copy before deploying:
ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112 \
  'sudo cp /usr/local/bin/dginf-coordinator /usr/local/bin/dginf-coordinator.bak'
```

To rollback:

```bash
ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112 \
  'sudo systemctl stop dginf-coordinator && \
   sudo mv /usr/local/bin/dginf-coordinator.bak /usr/local/bin/dginf-coordinator && \
   sudo systemctl start dginf-coordinator'
```

## Other services on this machine

- **nginx** — TLS termination and reverse proxy (443 → 8080). Config at `/etc/nginx/sites-enabled/dginf-mdm`.
- **MicroMDM** — Apple MDM server on port 9002 (`micromdm.service`). Used for device attestation.
- **step-ca** — ACME server for device certificate issuance (`step-ca.service`).
- **Let's Encrypt** — TLS certs at `/etc/letsencrypt/live/inference-test.openinnovation.dev/`.

## Troubleshooting

**Port 8080 already in use**: An old coordinator process is still running. Kill it manually:

```bash
sudo kill $(sudo lsof -ti :8080)
sudo systemctl start dginf-coordinator
```

**Service crash-looping**: Check logs:

```bash
sudo journalctl -u dginf-coordinator -n 50 --no-pager
```

**WebSocket disconnects**: The nginx config has `proxy_read_timeout 86400` (24h) for `/ws/`. If providers are disconnecting frequently, check nginx error logs:

```bash
sudo tail -50 /var/log/nginx/error.log
```

---

## Provider CLI & Bundle Distribution

Providers install via a curl one-liner that downloads a tarball from the coordinator server:

```bash
curl -fsSL https://inference-test.openinnovation.dev/install.sh | bash
```

This downloads `dginf-bundle-macos-arm64.tar.gz` from `/var/www/html/dl/` on the server and extracts it to `~/.dginf/`.

### What's in the bundle

| File | Description |
|------|-------------|
| `dginf-provider` | Rust CLI binary (arm64 macOS) |
| `dginf-enclave` | Swift Secure Enclave attestation helper |
| `python/` | Standalone Python 3.12 + mlx + vllm-mlx |

### Building the provider CLI

```bash
cd provider
PYO3_USE_ABI3_FORWARD_COMPATIBILITY=1 cargo build --release --no-default-features
```

The binary is at `target/release/dginf-provider`.

### Building the Secure Enclave helper

```bash
cd app/DGInf
swift build -c release --product dginf-enclave
```

### Creating the tarball bundle

The tarball bundles the provider binary, enclave helper, and a standalone Python environment with mlx/vllm-mlx pre-installed.

Prerequisites:
- Python bundle already set up at `~/.dginf/python/` (created by a prior install or manual setup)
- Provider binary built (`cargo build --release --no-default-features`)
- Enclave binary built (`swift build -c release`)

```bash
# Create bundle directory
mkdir -p /tmp/dginf-bundle
cp provider/target/release/dginf-provider /tmp/dginf-bundle/
cp app/DGInf/.build/release/dginf-enclave /tmp/dginf-bundle/
cp -a ~/.dginf/python /tmp/dginf-bundle/

# Create tarball
cd /tmp/dginf-bundle
tar czf dginf-bundle-macos-arm64.tar.gz dginf-provider dginf-enclave python/
```

### Uploading the bundle to the server

```bash
scp -i ~/.ssh/dginf-infra /tmp/dginf-bundle/dginf-bundle-macos-arm64.tar.gz \
  ubuntu@34.197.17.112:/tmp/

ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112 \
  'sudo mv /tmp/dginf-bundle-macos-arm64.tar.gz /var/www/html/dl/ && \
   ls -lh /var/www/html/dl/dginf-bundle-macos-arm64.tar.gz'
```

### Updating install.sh

The install script is at `scripts/install.sh` in the repo and served from `/var/www/html/install.sh` on the server. To update:

```bash
scp -i ~/.ssh/dginf-infra scripts/install.sh ubuntu@34.197.17.112:/tmp/ && \
ssh -i ~/.ssh/dginf-infra ubuntu@34.197.17.112 \
  'sudo mv /tmp/install.sh /var/www/html/install.sh'
```

### Verify the distribution

```bash
# Check tarball is accessible
curl -sI https://inference-test.openinnovation.dev/dl/dginf-bundle-macos-arm64.tar.gz | head -5

# Check install script is accessible
curl -s https://inference-test.openinnovation.dev/install.sh | head -5
```

---

## macOS App Bundle (.app / .dmg)

For distributing the full macOS menu bar app (not just the CLI).

### Build the app bundle

```bash
./scripts/bundle-app.sh                                      # Ad-hoc signing (testing only)
./scripts/bundle-app.sh "Developer ID Application: OrgName"  # Production signing
./scripts/bundle-app.sh "Developer ID Application: OrgName" --notarize  # + Apple notarization
```

This produces:
- `build/DGInf.app` — Code-signed macOS app with hardened runtime
- `build/DGInf-0.1.0.dmg` — Drag-and-drop installer DMG

The app bundle includes the provider binary, enclave helper, and bundled Python — all code-signed so any tampering breaks the signature.

### Prerequisites for the app bundle

```bash
# 1. Build the Rust provider (release, no Python feature to avoid linking issues)
cd provider && cargo build --release --no-default-features && cd ..

# 2. Build the Swift app and enclave helper
cd app/DGInf && swift build -c release && cd ../..

# 3. Ensure Python bundle exists at ~/.dginf/python/
```

### Distributing the DMG

Currently manual — upload the DMG wherever you want to host it. The install.sh flow uses the tarball, not the DMG. The DMG is for users who prefer a traditional macOS app install.

---

## Server file layout

Files served by nginx from `/var/www/html/`:

```
/var/www/html/
├── index.html                          # Landing page
├── install.sh                          # curl installer script
├── enroll.mobileconfig                 # MDM enrollment profile
├── dginf-provider-macos-arm64.tar.gz   # Legacy standalone provider tarball
└── dl/
    ├── dginf-bundle-macos-arm64.tar.gz # Full bundle (provider + python + enclave)
    ├── dginf-provider                  # Standalone provider binary
    └── dginf-enclave                   # Standalone enclave binary
```
