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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/main.go

# Stage 3: Final Runtime Image
FROM ubuntu:noble

# Cài đặt các gói cần thiết tối thiểu
# 'sudo' cần thiết để '--with-deps' có thể cài đặt các thư viện hệ thống
RUN apt-get update && apt-get install -y ca-certificates tzdata sudo && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy file binary ứng dụng và migrations
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Tạo user không phải root
RUN useradd --create-home --shell /bin/bash appuser
# Thêm user vào nhóm sudo để có thể chạy lệnh cài đặt với quyền cao hơn
RUN adduser appuser sudo
# Cấu hình sudo để không hỏi mật khẩu
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Cấp quyền cho user trên thư mục ứng dụng
RUN chown -R appuser:appuser /app

# ---- PHẦN THAY ĐỔI QUAN TRỌNG NHẤT ----
# Chuyển sang user 'appuser'
USER appuser


RUN npx playwright@latest install --with-deps chromium

# Expose port
EXPOSE 8080

# Lệnh chạy ứng dụng (vẫn đang là 'appuser')
CMD ["/app/main"]
