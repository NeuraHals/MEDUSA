FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /messaging-agent ./cmd/server

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /messaging-agent .
COPY --from=builder /app/config.yaml .
USER 65532:65532
EXPOSE 8086
ENTRYPOINT ["/messaging-agent"]
