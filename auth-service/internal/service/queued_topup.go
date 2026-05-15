package service

import (
	"fmt"

	"github.com/final-ap2-course2/auth-service/internal/messaging"
)

type QueuedTopUpHandler struct {
	auth *AuthService
	user *UserService
}

func NewQueuedTopUpHandler(auth *AuthService, user *UserService) *QueuedTopUpHandler {
	return &QueuedTopUpHandler{auth: auth, user: user}
}

func (h *QueuedTopUpHandler) ProcessQueuedTopUp(evt messaging.PendingTopUp) error {
	if evt.AccessToken != "" {
		actor, err := h.auth.ValidateToken(evt.AccessToken)
		if err != nil {
			return err
		}
		if actor.ID != evt.TargetUserID && actor.Role != "admin" {
			return fmt.Errorf("forbidden")
		}
	} else if evt.ActorUserID != "" {
		if evt.ActorUserID != evt.TargetUserID {
			return fmt.Errorf("forbidden")
		}
	} else {
		return fmt.Errorf("missing authorization")
	}
	return h.user.ApplyTopUpBalance(evt.TargetUserID, evt.Amount)
}
