FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o zte-cpe .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/zte-cpe /usr/local/bin/zte-cpe

EXPOSE 9101

ENV ZTE_TYPE=g5ts
ENV ZTE_URL=http://192.168.0.1
ENV ZTE_LISTEN=:9101
ENV ZTE_INTERVAL=30

ENTRYPOINT ["zte-cpe", "serve"]
