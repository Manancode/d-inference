FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o coordinator ./cmd/coordinator

# Download MicroMDM and step-ca binaries.
FROM alpine:3.21 AS fetcher
RUN apk add --no-cache curl tar
# MicroMDM v1.12.1
RUN curl -fsSL https://github.com/micromdm/micromdm/releases/download/v1.12.1/micromdm_v1.12.1_linux_amd64.tar.gz \
    | tar -xz -C /tmp && mv /tmp/build/linux/micromdm /usr/local/bin/micromdm
# step-ca (Smallstep)
RUN curl -fsSL https://dl.smallstep.com/gh-release/certificates/gh-release-header/v0.28.3/step-ca_linux_0.28.3_amd64.tar.gz \
    | tar -xz -C /tmp && mv /tmp/step-ca_0.28.3/bin/step-ca /usr/local/bin/step-ca
RUN curl -fsSL https://dl.smallstep.com/gh-release/cli/gh-release-header/v0.28.3/step_linux_0.28.3_amd64.tar.gz \
    | tar -xz -C /tmp && mv /tmp/step_0.28.3/bin/step /usr/local/bin/step

FROM alpine:3.21
RUN apk add --no-cache ca-certificates postgresql16 postgresql16-contrib su-exec

# Binaries.
COPY --from=builder /app/coordinator /usr/local/bin/coordinator
COPY --from=fetcher /usr/local/bin/micromdm /usr/local/bin/micromdm
COPY --from=fetcher /usr/local/bin/step-ca /usr/local/bin/step-ca
COPY --from=fetcher /usr/local/bin/step /usr/local/bin/step

# Static files served by Caddy (install script, bundles, enrollment profile).
RUN mkdir -p /var/www/html/dl

# Persistent data directories.
ENV PGDATA=/data/postgres
RUN mkdir -p /data/postgres /data/micromdm /data/step-ca /run/postgresql && \
    chown -R postgres:postgres /data/postgres /run/postgresql

# Coordinator config defaults.
ENV EIGENINFERENCE_PORT=8080
ENV EIGENINFERENCE_DATABASE_URL=postgres://coordinator:coordinator@localhost:5432/eigeninference?sslmode=disable
ENV EIGENINFERENCE_MDM_URL=https://localhost:9002
ENV EIGENINFERENCE_MDM_API_KEY=eigeninference-micromdm-api
ENV EIGENINFERENCE_STEP_CA_ROOT=/data/step-ca/certs/root_ca.crt
ENV EIGENINFERENCE_STEP_CA_INTERMEDIATE=/data/step-ca/certs/intermediate_ca.crt

COPY <<'STARTSCRIPT' /usr/local/bin/start.sh
#!/bin/sh
set -e

# ---- PostgreSQL ----
if [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "Initializing PostgreSQL..."
    su-exec postgres initdb -D "$PGDATA" --auth=trust --no-locale --encoding=UTF8
    echo "listen_addresses = '127.0.0.1'" >> "$PGDATA/postgresql.conf"
    echo "unix_socket_directories = '/run/postgresql'" >> "$PGDATA/postgresql.conf"
fi
su-exec postgres pg_ctl start -D "$PGDATA" -l /data/postgres.log -w
su-exec postgres psql -h /run/postgresql -tc \
    "SELECT 1 FROM pg_roles WHERE rolname='coordinator'" | grep -q 1 || \
    su-exec postgres psql -h /run/postgresql -c "CREATE ROLE coordinator WITH LOGIN PASSWORD 'coordinator'"
su-exec postgres psql -h /run/postgresql -tc \
    "SELECT 1 FROM pg_database WHERE datname='eigeninference'" | grep -q 1 || \
    su-exec postgres psql -h /run/postgresql -c "CREATE DATABASE eigeninference OWNER coordinator"
echo "PostgreSQL ready."

# ---- step-ca ----
if [ -d "/data/step-ca/config" ]; then
    echo "Starting step-ca..."
    step-ca /data/step-ca/config/ca.json \
        --password-file /data/step-ca/secrets/password \
        >> /data/step-ca.log 2>&1 &
    echo "step-ca started (port 9000)."
else
    echo "step-ca not initialized — skipping. Run 'step ca init' to set up."
fi

# ---- MicroMDM ----
if [ -d "/data/micromdm" ] && [ -n "$MICROMDM_API_KEY" ]; then
    echo "Starting MicroMDM..."
    micromdm serve \
        -server-url "https://${DOMAIN:-localhost}" \
        -api-key "${MICROMDM_API_KEY:-eigeninference-micromdm-api}" \
        -config-path /data/micromdm \
        -tls-cert /data/micromdm/server.crt \
        -tls-key /data/micromdm/server.key \
        -http-addr :9002 \
        >> /data/micromdm.log 2>&1 &
    echo "MicroMDM started (port 9002)."
else
    echo "MicroMDM not configured — skipping. Set MICROMDM_API_KEY and provide certs."
fi

# ---- Coordinator (PID 1 — receives SIGTERM from EigenCloud) ----
echo "Starting coordinator..."
exec coordinator
STARTSCRIPT

RUN chmod +x /usr/local/bin/start.sh

EXPOSE 8080
CMD ["start.sh"]
