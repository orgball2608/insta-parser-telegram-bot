FROM golang:1.21-alpine3.19 as be-builder
ARG BE_PATH
RUN echo "BE_PATH ${BE_PATH}"
ENV GO111MODULE=on

WORKDIR /app

RUN apk add --no-cache git curl wget upx make

COPY ${BE_PATH}go.mod ${BE_PATH}go.sum ./

RUN go mod download

COPY ${BE_PATH}. .

RUN make build

# Start a new stage from scratch
FROM alpine:latest

# RUN apk --no-cache add ca-certificates
RUN apk --no-cache add tzdata

WORKDIR /app

COPY --from=be-builder /app/build-out /app/
COPY --from=be-builder /app/docker-entrypoint.sh /app/
COPY --from=be-builder /app/Makefile /app/

RUN chmod +x /app/docker-entrypoint.sh

CMD ["sh", "/app/docker-entrypoint.sh"]
