FROM golang:1.23-alpine AS builder

WORKDIR /app

# 复制go mod和sum文件
COPY go.mod go.sum* ./

# 设置Go代理
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o email-service .

# 使用轻量级镜像
FROM alpine:latest

WORKDIR /app

# 安装依赖
RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 从builder阶段复制编译好的应用
COPY --from=builder /app/email-service .

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

# 暴露端口
EXPOSE ${PORT:-8080}

# 运行应用
CMD ["./email-service"] 