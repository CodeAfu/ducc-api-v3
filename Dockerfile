FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    curl \
    libreoffice-writer-nogui \
    fonts-liberation \
    fontconfig \
    python3 \
    python3-venv \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Run Stage
# FROM scratch

WORKDIR /app

COPY --from=builder /app/server /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY scripts/ /app/scripts/
COPY templates/ /app/templates/
# COPY templates/fonts/ usr/share/fonts/custom/
RUN pip3 install --break-system-packages --no-cache-dir -r /app/scripts/requirements.txt
RUN fc-cache -fv

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8088/api/v3/health || exit 1

EXPOSE 8088

ENTRYPOINT ["/app/server"]
