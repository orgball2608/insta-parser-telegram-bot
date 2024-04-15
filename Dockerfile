FROM golang:1.20-alpine3.19 as be-builder
ARG BE_PATH
RUN echo "BE_PATH ${BE_PATH}"
ENV GO111MODULE=on

WORKDIR /app

COPY ${BE_PATH}go.mod ${BE_PATH}go.sum ./

RUN go mod download

COPY ${BE_PATH}. .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/main.go

# Start a new stage from scratch
FROM alpine:latest

# RUN apk --no-cache add ca-certificates
#RUN apk --no-cache add ca-certificates tzdata

# Done apk add

WORKDIR /root/

# RUN apk --no-cache add ca-certificates
# RUN apk add --no-cache tzdata

COPY --from=be-builder /app/app ./

# Run image

CMD ["./app"]
