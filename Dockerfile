# Stage 1: Go Modules Caching
FROM golang:1.22-alpine AS modules
WORKDIR /modules
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

# Stage 2: Builder
FROM golang:1.22-alpine AS builder
COPY --from=modules /go/pkg /go/pkg
WORKDIR /app
COPY . .
ENV GOTOOLCHAIN=auto
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/main.go


RUN apk add --no-cache go
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.0


FROM mcr.microsoft.com/playwright:v1.52.0-jammy

ENV DEBIAN_FRONTEND=noninteractive

# Install Go runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates tzdata sudo curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app


COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /go/bin/playwright /usr/local/bin/playwright
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

RUN useradd --create-home --shell /bin/bash appuser || true
RUN usermod -aG sudo appuser || true
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers


RUN chown -R appuser:appuser /app

USER appuser
RUN /usr/local/bin/playwright install --with-deps

# Expose port
EXPOSE 8080

# Set the entrypoint script to configure environment variables
ENTRYPOINT ["/docker-entrypoint.sh"]


CMD ["/app/main"]
