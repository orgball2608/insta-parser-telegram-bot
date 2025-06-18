# Stage 1: Go Modules Caching
FROM golang:1.23.4-alpine AS modules
WORKDIR /modules
COPY go.mod go.sum ./
RUN go mod download

# Stage 2: Builder
FROM golang:1.23.4-alpine AS builder

COPY --from=modules /go/pkg /go/pkg

WORKDIR /app
COPY . .

# Tự động lấy đúng phiên bản Playwright từ go.mod và cài đặt CLI
RUN PW_VERSION=$(go list -m -f '{{.Version}}' github.com/playwright-community/playwright-go) && \
    go install github.com/playwright-community/playwright-go/cmd/playwright@${PW_VERSION}

# Build ứng dụng Go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/main.go

# Stage 3: Final Runtime Image
# Bắt đầu từ một base image Ubuntu sạch sẽ. 'noble' là Ubuntu 24.04.
FROM ubuntu:noble

# Cài đặt các chứng chỉ CA và timezone
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy các file cần thiết từ stage builder
COPY --from=builder /go/bin/playwright /usr/local/bin/
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Chạy lệnh install của Playwright để tải trình duyệt VÀ các thư viện hệ thống cần thiết
RUN playwright install --with-deps chromium

# Tạo một user không phải root để chạy ứng dụng
RUN useradd --create-home --shell /bin/bash appuser
RUN chown -R appuser:appuser /app
USER appuser

# Expose port ứng dụng
EXPOSE 8080

# Lệnh chạy ứng dụng
CMD ["/app/main"]
