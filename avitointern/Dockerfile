FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o avitointern cmd/avitointern/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/avitointern .
COPY --from=builder /app/static ./static
COPY --from=builder /app/.env ./

RUN apk add --no-cache bash

EXPOSE 8080

CMD ["./avitointern"]
