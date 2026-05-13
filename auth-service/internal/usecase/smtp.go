package usecase

import (
	"fmt"
	"log"
)

func SendResetEmail(email, resetToken string) error {
	log.Printf("Sending password reset email to: %s", email)
	log.Printf("Reset token: %s", resetToken)
	resetLink := fmt.Sprintf("https://yourapp.com/reset-password?token=%s", resetToken)
	log.Printf("Password reset link: %s", resetLink)
	return nil
}

func SendWelcomeEmail(email, username string) error {
	log.Printf("Sending welcome email to: %s (username: %s)", email, username)
	return nil
}

func SendPasswordChangedEmail(email string) error {
	log.Printf("Sending password changed notification to: %s", email)
	return nil
}
