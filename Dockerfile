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

RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Tạo user không phải root NGAY TỪ ĐẦU
RUN useradd --create-home --shell /bin/bash appuser

# Copy file binary và migrations vào thư mục app
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Cấp quyền cho user mới trên thư mục ứng dụng
RUN chown -R appuser:appuser /app

USER appuser

# Chạy lệnh install của Playwright với quyền của 'appuser'
# Driver sẽ được cài vào /home/appuser/.cache/ms-playwright
RUN playwright install --with-deps chromium

# Expose port ứng dụng
EXPOSE 8080

# Lệnh chạy ứng dụng
CMD ["/app/main"]
