# Stage 1: 前端打包
FROM node:22-alpine AS frontend
WORKDIR /app
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build
RUN ls -la dist/ && cp -r dist /out-dist

# Stage 2: Go 编译
FROM golang:1.25-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 把前端产物放到 internal/embed/dist 让 go:embed 找到
COPY --from=frontend /out-dist ./internal/embed/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo docker)" \
    -o /out/leaderboard ./cmd/leaderboard

# Stage 3: 运行时
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
WORKDIR /app
COPY --from=backend /out/leaderboard .
RUN mkdir -p /app/data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -q --spider http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/leaderboard"]
