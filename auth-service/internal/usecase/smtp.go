package usecase

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func loadSMTPConfig() SMTPConfig {
	host := getEnv("SMTP_HOST", "smtp.gmail.com")
	return SMTPConfig{
		Host:     host,
		Port:     getEnv("SMTP_PORT", "587"),
		Username: getEnv("SMTP_USERNAME", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", getEnv("SMTP_USERNAME", "noreply@example.com")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func sendEmail(to, subject, body string) error {
	cfg := loadSMTPConfig()

	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		if cfg.Host != "mailhog" {
			log.Printf("[SMTP] credentials not configured — logging email instead")
			log.Printf("[SMTP] TO: %s | SUBJECT: %s | BODY: %s", to, subject, body)
			return nil
		}
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	headers := map[string]string{
		"From":          cfg.From,
		"To":            to,
		"Subject":       subject,
		"MIME-Version":  "1.0",
		"Content-Type":  "text/plain; charset=UTF-8",
	}
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k + ": " + v + "\r\n")
	}
	sb.WriteString("\r\n" + body)
	msg := []byte(sb.String())

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		if os.Getenv("SMTP_TLS_INSECURE") == "true" {
			tlsCfg.InsecureSkipVerify = true
		}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if cfg.Username != "" && cfg.Password != "" {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer wc.Close()
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	log.Printf("[SMTP] email sent to %s — subject: %s", to, subject)
	return nil
}

func SendResetEmail(email, resetToken string) error {
	appURL := strings.TrimSuffix(getEnv("APP_URL", "https://yourapp.com"), "/")
	resetLink := fmt.Sprintf("%s/auth.html?token=%s", appURL, resetToken)

	subject := "Password Reset Request"
	body := fmt.Sprintf(`Hello,

You requested a password reset for your account.

Click the link below to reset your password (valid for 1 hour):
%s

If you did not request this, please ignore this email.

— The E-Commerce Team
`, resetLink)

	if err := sendEmail(email, subject, body); err != nil {
		log.Printf("[SMTP] SendResetEmail failed: %v", err)
		return err
	}
	return nil
}

func SendWelcomeEmail(email, username string) error {
	subject := "Welcome to E-Commerce!"
	body := fmt.Sprintf(`Hi %s,

Your account has been created successfully. Welcome aboard!

You can now log in and start shopping.

— The E-Commerce Team
`, username)

	if err := sendEmail(email, subject, body); err != nil {
		log.Printf("[SMTP] SendWelcomeEmail failed: %v", err)
		return err
	}
	return nil
}

func SendPasswordChangedEmail(email string) error {
	subject := "Your Password Was Changed"
	body := `Hello,

Your password has been changed successfully.

If you did not make this change, please contact support immediately.

— The E-Commerce Team
`
	if err := sendEmail(email, subject, body); err != nil {
		log.Printf("[SMTP] SendPasswordChangedEmail failed: %v", err)
		return err
	}
	return nil
}

func SendPasswordChangeVerificationEmail(email, code string) error {
	subject := "Your password change verification code"
	body := fmt.Sprintf(`Hello,

You requested to change your password. Use this verification code (valid for 15 minutes):

%s

If you did not request this, please ignore this email and your password will stay the same.

— The E-Commerce Team
`, code)
	if err := sendEmail(email, subject, body); err != nil {
		log.Printf("[SMTP] SendPasswordChangeVerificationEmail failed: %v", err)
		return err
	}
	return nil
}
