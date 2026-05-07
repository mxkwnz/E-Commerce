package usecase

import "log"

func SendResetEmail(email string) error {
	log.Printf("Sending password reset email to: %s", email)
	// TODO: Implement actual SMTP logic
	return nil
}
