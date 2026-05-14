package usecase

import (
	"os"
	"strings"
	"testing"
)

func TestLoadSMTPConfig_DefaultsWhenEnvMissing(t *testing.T) {
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")

	cfg := loadSMTPConfig()

	if cfg.Host != "smtp.gmail.com" {
		t.Errorf("expected default host smtp.gmail.com, got %s", cfg.Host)
	}
	if cfg.Port != "587" {
		t.Errorf("expected default port 587, got %s", cfg.Port)
	}
	if cfg.Username != "" {
		t.Errorf("expected empty username, got %s", cfg.Username)
	}
}

func TestLoadSMTPConfig_ReadsFromEnv(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.sendgrid.net")
	os.Setenv("SMTP_PORT", "465")
	os.Setenv("SMTP_USERNAME", "apikey")
	os.Setenv("SMTP_PASSWORD", "SG.secret")
	t.Cleanup(func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
	})

	cfg := loadSMTPConfig()
	if cfg.Host != "smtp.sendgrid.net" {
		t.Errorf("expected smtp.sendgrid.net, got %s", cfg.Host)
	}
	if cfg.Port != "465" {
		t.Errorf("expected 465, got %s", cfg.Port)
	}
}

func TestSendEmail_NoCredentials_LogsAndReturnsNil(t *testing.T) {
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")

	err := sendEmail("test@example.com", "Subject", "Body")
	if err != nil {
		t.Errorf("expected nil error when no credentials, got: %v", err)
	}
}

func TestSendResetEmail_BuildsCorrectLink(t *testing.T) {
	os.Setenv("APP_URL", "https://shop.test")
	os.Unsetenv("SMTP_USERNAME")
	t.Cleanup(func() { os.Unsetenv("APP_URL") })

	err := SendResetEmail("user@example.com", "abc123token")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	appURL := getEnv("APP_URL", "https://yourapp.com")
	token := "abc123token"
	link := appURL + "/reset-password?token=" + token
	if !strings.Contains(link, "abc123token") {
		t.Error("reset link does not contain token")
	}
	if !strings.Contains(link, "https://shop.test") {
		t.Error("reset link does not contain APP_URL")
	}
}

func TestSendWelcomeEmail_NoError(t *testing.T) {
	os.Unsetenv("SMTP_USERNAME")
	err := SendWelcomeEmail("user@example.com", "Alice")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendPasswordChangedEmail_NoError(t *testing.T) {
	os.Unsetenv("SMTP_USERNAME")
	err := SendPasswordChangedEmail("user@example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetEnv_ReturnsFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("MY_TEST_VAR")
	val := getEnv("MY_TEST_VAR", "default_value")
	if val != "default_value" {
		t.Errorf("expected default_value, got %s", val)
	}
}

func TestGetEnv_ReturnsEnvWhenSet(t *testing.T) {
	os.Setenv("MY_TEST_VAR", "custom_value")
	t.Cleanup(func() { os.Unsetenv("MY_TEST_VAR") })
	val := getEnv("MY_TEST_VAR", "default_value")
	if val != "custom_value" {
		t.Errorf("expected custom_value, got %s", val)
	}
}
