FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o exporter ./main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/exporter /exporter
COPY --from=builder /app/config.yaml /config.yaml

EXPOSE 9877

CMD ["/exporter"]
