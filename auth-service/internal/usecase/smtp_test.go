package usecase

import "testing"

func TestSendResetEmail_NoSMTPConfigured(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	if err := SendResetEmail("user@example.com", "abc"); err != nil {
		t.Fatal(err)
	}
}
