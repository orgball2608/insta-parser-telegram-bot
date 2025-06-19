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

# Install runtime dependencies and Go 1.22
RUN apt-get update && apt-get install -y ca-certificates tzdata sudo curl wget && rm -rf /var/lib/apt/lists/*
RUN wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz && \
    rm go1.22.4.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app


COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

RUN useradd --create-home --shell /bin/bash appuser || true
RUN usermod -aG sudo appuser || true
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers


RUN chown -R appuser:appuser /app

# Cài playwright CLI trong stage cuối
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.0
RUN install -m 0755 /root/go/bin/playwright /usr/local/bin/playwright

USER appuser
RUN playwright install --with-deps

# Expose port
EXPOSE 8080

# Set the entrypoint script to configure environment variables
ENTRYPOINT ["/docker-entrypoint.sh"]


CMD ["/app/main"]
