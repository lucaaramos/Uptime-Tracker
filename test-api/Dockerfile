FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /test-api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /test-api /usr/local/bin/test-api

EXPOSE 9000

CMD ["test-api"]
