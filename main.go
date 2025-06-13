package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
)

// Config 存储应用配置
type Config struct {
	Port         string
	WebhookToken string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	EmailFrom    string
}

// EmailPayload 定义邮件请求负载结构
type EmailPayload struct {
	To          []string               `json:"to"`
	Cc          []string               `json:"cc,omitempty"`
	Bcc         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	IsHtml      bool                   `json:"is_html,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Attachment 定义邮件附件
type Attachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // Base64编码的内容
}

var config Config

// 加载配置
func loadConfig() error {
	// 尝试加载.env文件，但不强制要求
	_ = godotenv.Load()

	config.Port = getEnv("PORT", "8080")
	config.WebhookToken = getEnv("WEBHOOK_TOKEN", "")
	config.SMTPHost = getEnv("SMTP_HOST", "")
	config.SMTPPort = getEnvAsInt("SMTP_PORT", 587)
	config.SMTPUser = getEnv("SMTP_USER", "")
	config.SMTPPass = getEnv("SMTP_PASS", "")
	config.EmailFrom = getEnv("EMAIL_FROM", "")

	// 验证必要的配置
	if config.WebhookToken == "" {
		return fmt.Errorf("必须设置WEBHOOK_TOKEN")
	}
	if config.SMTPHost == "" || config.SMTPUser == "" || config.SMTPPass == "" {
		return fmt.Errorf("必须设置SMTP相关配置")
	}
	if config.EmailFrom == "" {
		return fmt.Errorf("必须设置默认发件人邮箱")
	}

	return nil
}

// 获取环境变量，如果不存在则使用默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value := 0
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		log.Printf("警告: 无法解析环境变量 %s 的值 '%s' 为整数, 使用默认值 %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}

// 验证请求头中的token
func validateToken(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// 支持Bearer token或直接token方式
	token := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	return token == config.WebhookToken
}

// 发送邮件
func sendEmail(payload EmailPayload) error {
	// 创建邮件消息
	m := gomail.NewMessage()
	m.SetHeader("From", config.EmailFrom)

	// 设置收件人
	if len(payload.To) == 0 {
		return fmt.Errorf("收件人不能为空")
	}
	m.SetHeader("To", payload.To...)

	// 设置抄送
	if len(payload.Cc) > 0 {
		m.SetHeader("Cc", payload.Cc...)
	}

	// 设置密送
	if len(payload.Bcc) > 0 {
		m.SetHeader("Bcc", payload.Bcc...)
	}

	// 设置主题
	m.SetHeader("Subject", payload.Subject)

	// 设置正文
	contentType := "text/plain"
	if payload.IsHtml {
		contentType = "text/html"
	}
	m.SetBody(contentType, payload.Body)

	// 添加自定义头部
	for key, value := range payload.Headers {
		m.SetHeader(key, value)
	}

	// TODO: 处理附件 (需要Base64解码)
	// 这里暂不实现附件功能，可以根据需要扩展

	// 配置SMTP发送器
	d := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass)

	// 发送邮件
	return d.DialAndSend(m)
}

// 邮件发送webhook处理函数
func emailWebhookHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 验证token
	if !validateToken(r) {
		http.Error(w, "未授权访问", http.StatusUnauthorized)
		log.Println("未授权的API访问尝试")
		return
	}

	// 解码JSON请求体
	var payload EmailPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "无效的请求体", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	// 发送邮件
	err = sendEmail(payload)

	// 处理响应
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		log.Printf("发送邮件失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": fmt.Sprintf("邮件发送失败: %v", err),
		})
		return
	}

	// 计算处理时间
	duration := time.Since(startTime).Milliseconds()

	// 记录发送成功日志
	log.Printf("邮件已发送: 主题=%s, 收件人=%v, 处理时间=%dms",
		payload.Subject, payload.To, duration)

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "邮件已发送",
		"metadata": map[string]interface{}{
			"duration_ms": duration,
			"recipients":  len(payload.To) + len(payload.Cc) + len(payload.Bcc),
		},
	})
}

// 健康检查处理函数
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0.0"})
}

func main() {
	// 加载配置
	err := loadConfig()
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}

	// 设置路由
	http.HandleFunc("/send", emailWebhookHandler)
	http.HandleFunc("/health", healthHandler)

	// 启动服务器
	serverAddr := ":" + config.Port
	log.Printf("邮件发送服务启动在 %s", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
