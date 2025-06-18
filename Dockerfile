# Stage 1: Go Modules Caching
# Tách riêng việc download module để tận dụng Docker cache.
# Nếu go.mod/go.sum không đổi, bước này sẽ không chạy lại.
FROM golang:1.23.4-alpine AS modules
WORKDIR /modules
COPY go.mod go.sum ./
RUN go mod download

# Stage 2: Builder
# Build ứng dụng và chuẩn bị các file cần thiết.
FROM golang:1.21-alpine AS builder

# Copy các module đã được cache từ stage trước
COPY --from=modules /go/pkg /go/pkg

WORKDIR /app
COPY . .

# Tự động lấy đúng phiên bản Playwright từ go.mod và cài đặt CLI
# Điều này đảm bảo phiên bản CLI và thư viện luôn khớp nhau.
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
COPY --from=builder /go/bin/playwright /usr/local/bin/  # Copy Playwright CLI
COPY --from=builder /app/main .                        # Copy file binary của bạn
COPY --from=builder /app/migrations ./migrations      # Copy migrations

# Chạy lệnh install của Playwright để tải trình duyệt VÀ các thư viện hệ thống cần thiết.
# '--with-deps' là cờ quan trọng nhất ở đây.
RUN playwright install --with-deps chromium # Chỉ cài chromium để image nhẹ hơn
# Nếu bạn cần các trình duyệt khác, dùng: playwright install --with-deps

# Tạo một user không phải root để chạy ứng dụng
RUN useradd --create-home --shell /bin/bash appuser
# Cấp quyền cho user mới trên thư mục ứng dụng
RUN chown -R appuser:appuser /app
# Chuyển sang user mới
USER appuser

# Expose port ứng dụng
EXPOSE 8080

# Lệnh chạy ứng dụng
CMD ["/app/main"]
