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

# Cài đặt Playwright CLI vào một nơi có thể truy cập được
# Dùng `go list` để lấy đúng phiên bản từ go.mod
RUN PW_VERSION=$(go list -m -f '{{.Version}}' github.com/playwright-community/playwright-go) && \
    go install github.com/playwright-community/playwright-go/cmd/playwright@${PW_VERSION}

# Build ứng dụng
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/main.go

# Stage 3: Final Runtime Image
FROM ubuntu:noble

# Cài đặt các phụ thuộc cơ bản
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# ---- PHẦN SỬA LỖI QUAN TRỌNG ----
# Copy Playwright CLI từ stage builder sang stage cuối cùng
COPY --from=builder /go/bin/playwright /usr/local/bin/

# Copy file binary ứng dụng và migrations
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Chạy lệnh install để tải trình duyệt VÀ các thư viện hệ thống cần thiết
# Lệnh '--with-deps' sẽ tự động `apt-get install` các thư viện đồ họa
RUN playwright install --with-deps chromium

# Tạo user không phải root
RUN useradd --create-home --shell /bin/bash appuser
# Cấp quyền cho user mới trên thư mục ứng dụng
RUN chown -R appuser:appuser /app
# Chuyển sang user mới
USER appuser

# Expose port
EXPOSE 8080

# Lệnh chạy ứng dụng
CMD ["/app/main"]
