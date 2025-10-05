# ---- builder ----
FROM golang:1.25 AS builder
ENV CGO_ENABLED=1 GO111MODULE=on GOTOOLCHAIN=auto
WORKDIR /src

COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -c 'go mod download || true'

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags="-s -w" -o /bin/gotrack ./cmd/gotrack

# ---- runner ----
FROM gcr.io/distroless/base-debian12:nonroot AS runner
WORKDIR /app
USER nonroot:nonroot
COPY --from=builder /bin/gotrack /app/gotrack

# Create directory for SSL certificates (if needed)
# Note: SSL certificates should be mounted as volumes in production
# Example: docker run -v /path/to/certs:/app/certs gotrack

EXPOSE 19890
ENTRYPOINT ["/app/gotrack"]
