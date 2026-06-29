# syntax=docker/dockerfile:1

FROM node:22-alpine AS web
WORKDIR /src/web/ui
COPY web/ui/package.json web/ui/package-lock.json* ./
RUN npm ci || npm install
COPY web/ui/ ./
RUN npm run build

FROM golang:1.25-bookworm AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0
ENV GOMAXPROCS=1
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist /src/web/dist
RUN go build -o /flowagent ./cmd/flowagent

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates ffmpeg curl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /flowagent /usr/local/bin/flowagent
COPY config/ ./config/
COPY docs/workflows/ ./docs/workflows/
COPY --from=web /src/web/dist ./web/dist
ENV FLOWAGENT_BIND=0.0.0.0:8080
ENV FLOWAGENT_DATA_DIR=/data
ENV FLOWAGENT_AUTH_ENABLED=true
ENV FLOWAGENT_SMS_PROVIDER=dev
ENV FLOWAGENT_SMS_DEV_CODE=123456
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8080/api/health || exit 1
CMD ["flowagent", "serve", "--addr", "0.0.0.0:8080"]
