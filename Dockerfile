# --- STAGE 1: Build a static Go binary ---
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/main.go


# --- STAGE 2: Create the final runtime image ---
# Use the official Microsoft Playwright image which includes all browsers and dependencies.
# Match the image tag to your playwright-go library version.
FROM mcr.microsoft.com/playwright/go:v1.45.1-jammy

WORKDIR /app

# Copy the pre-built binary and migrations from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Grant execution permissions to the non-root 'pwuser'
USER root
RUN chown pwuser:pwuser /app/main && chmod +x /app/main

# Switch to the non-root user for security
USER pwuser

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["/app/main"]
