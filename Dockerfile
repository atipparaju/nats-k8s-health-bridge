# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install ca-certificates for TLS support in final image
RUN apk add --no-cache ca-certificates git

WORKDIR /build

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -trimpath \
    -o /nats-k8s-health-bridge \
    ./cmd/sidecar

# Stage 2: Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CA certs for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary and default config
COPY --from=builder /nats-k8s-health-bridge /nats-k8s-health-bridge
COPY --from=builder /build/config.yaml /etc/nats-k8s-health-bridge/config.yaml

# Run as non-root (nonroot user is uid 65532 in distroless)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/nats-k8s-health-bridge"]
