# 邮件发送微服务

这是一个基于Go语言开发的邮件发送微服务，它通过webhook接口接收邮件内容并发送邮件，可以作为各种系统的邮件发送组件。

## 功能特点

- 提供简单的API接口发送邮件
- 支持普通文本和HTML格式邮件
- 支持抄送(CC)和密送(BCC)
- 支持自定义邮件头部
- 通过Authorization头进行请求验证
- 支持Docker部署
- 健康检查接口
- 环境变量配置
- 可选的TLS证书验证跳过（解决自签名证书问题）

## 配置说明

服务通过环境变量进行配置，主要配置项如下：

| 环境变量 | 说明 | 默认值 |
|---------|------|-------|
| PORT | 服务监听端口 | 8080 |
| WEBHOOK_TOKEN | API验证令牌 | - |
| SMTP_HOST | SMTP服务器地址 | - |
| SMTP_PORT | SMTP服务器端口 | 587 |
| SMTP_USER | SMTP用户名 | - |
| SMTP_PASS | SMTP密码 | - |
| EMAIL_FROM | 默认发件人邮箱 | - |
| SKIP_TLS_VERIFY | 是否跳过TLS证书验证 | true |

可以通过`.env`文件或直接设置环境变量来配置。

## 本地运行

1. 克隆项目
2. 复制`env.example`为`.env`并修改配置
3. 运行服务：
   ```bash
   go run main.go
   ```

## Docker部署

### 构建镜像

```bash
docker build -t email-service:latest .
```

### 运行容器

```bash
docker run -p 8080:8080 --env-file .env email-service:latest
```

或使用环境变量：

```bash
docker run -p 8080:8080 \
  -e WEBHOOK_TOKEN=your_token \
  -e SMTP_HOST=smtp.example.com \
  -e SMTP_PORT=587 \
  -e SMTP_USER=user@example.com \
  -e SMTP_PASS=password \
  -e EMAIL_FROM=no-reply@example.com \
  -e SKIP_TLS_VERIFY=true \
  email-service:latest
```

## 常见问题

### TLS证书验证错误

如果遇到以下错误：
```
邮件发送失败: tls: failed to verify certificate: x509: certificate signed by unknown authority
```

这通常是因为SMTP服务器使用的是自签名证书或者内部CA签发的证书。解决方法：

1. 将`SKIP_TLS_VERIFY`环境变量设置为`true`
2. 这将跳过TLS证书验证，但请注意这会降低连接的安全性

## API接口

### POST /send

发送邮件接口。

#### 请求头

```
Authorization: your_token_here
```
或
```
Authorization: Bearer your_token_here
```

#### 请求体示例(JSON格式)

```json
{
  "to": ["recipient@example.com", "another@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "subject": "这是一封测试邮件",
  "body": "你好，这是邮件正文内容。",
  "is_html": false,
  "headers": {
    "X-Priority": "1",
    "X-Custom-Header": "Custom Value"
  },
  "metadata": {
    "user_id": 12345,
    "order_id": "ORD-20230613-001"
  }
}
```

#### 响应示例

成功：
```json
{
  "status": "success",
  "message": "邮件已发送",
  "metadata": {
    "duration_ms": 235,
    "recipients": 3
  }
}
```

失败：
```json
{
  "status": "error",
  "message": "邮件发送失败: 收件人不能为空"
}
```

### GET /health

健康检查接口，返回服务状态。 