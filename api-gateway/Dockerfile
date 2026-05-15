FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY telemetry /src/telemetry
WORKDIR /src/api-gateway
COPY api-gateway/go.mod api-gateway/go.sum ./
RUN go mod download
COPY api-gateway .
RUN go build -o api-gateway ./cmd/api-gateway

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /src/api-gateway/api-gateway .
EXPOSE 8080
CMD ["./api-gateway"]
