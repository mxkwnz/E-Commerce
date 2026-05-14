package usecase

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
)

func smtpDialAddr() string {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return ""
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	return net.JoinHostPort(host, port)
}

func smtpPlainAuth(host string) smtp.Auth {
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	if strings.TrimSpace(user) == "" && strings.TrimSpace(pass) == "" {
		return nil
	}
	return smtp.PlainAuth("", user, pass, host)
}

func sendMail(host, addr, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	helo := strings.TrimSpace(os.Getenv("SMTP_HELO_DOMAIN"))
	if helo == "" {
		helo = "localhost"
	}
	if err := c.Hello(helo); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: host}
		if os.Getenv("SMTP_TLS_INSECURE") == "true" {
			cfg.InsecureSkipVerify = true
		}
		if err := c.StartTLS(cfg); err != nil {
			return err
		}
	}
	if a := smtpPlainAuth(host); a != nil {
		if err := c.Auth(a); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func sendPlainEmail(to []string, subject, body string) error {
	addr := smtpDialAddr()
	if addr == "" {
		return nil
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = "noreply@localhost"
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	for _, rcpt := range to {
		fmt.Fprintf(&buf, "To: %s\r\n", rcpt)
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&buf, "\r\n%s\r\n", body)
	return sendMail(host, addr, from, to, buf.Bytes())
}

func resetLink(token string) string {
	base := strings.TrimSpace(os.Getenv("PASSWORD_RESET_BASE_URL"))
	if base == "" {
		base = "https://yourapp.com/reset-password"
	}
	base = strings.TrimSuffix(base, "/")
	return base + "?token=" + token
}

func SendResetEmail(email, resetToken string) error {
	body := fmt.Sprintf("Use this link to reset your password:\n\n%s\n\nIf you did not request a reset, ignore this message.\n", resetLink(resetToken))
	return sendPlainEmail([]string{email}, "Password reset", body)
}

func SendWelcomeEmail(email, username string) error {
	name := strings.TrimSpace(username)
	if name == "" {
		name = "there"
	}
	body := fmt.Sprintf("Hello %s,\n\nYour account was created successfully.\n", name)
	return sendPlainEmail([]string{email}, "Welcome", body)
}

func SendPasswordChangedEmail(email string) error {
	body := "Your password was changed. If this was not you, contact support immediately.\n"
	return sendPlainEmail([]string{email}, "Password changed", body)
}
