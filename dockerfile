FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o exporter ./main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/exporter /app/exporter
COPY config.yaml /app/config.yaml

EXPOSE 9877

CMD ["/app/exporter"]
