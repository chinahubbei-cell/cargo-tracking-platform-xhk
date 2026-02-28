# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 复制构建产物
COPY --from=builder /app/main .

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 5050

CMD ["./main"]
