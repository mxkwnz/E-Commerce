package messaging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	topupStream  = "BALANCE"
	topupSubject = "balance.topup.pending"
	topupDurable = "auth-topup-worker"
)

type TopUpProcessor interface {
	ProcessQueuedTopUp(evt PendingTopUp) error
}

type PendingTopUp struct {
	TargetUserID string  `json:"targetUserId"`
	AccessToken  string  `json:"accessToken"`
	ActorUserID  string  `json:"actorUserId,omitempty"`
	Amount       float64 `json:"amount"`
	QueuedAt     string  `json:"queuedAt"`
}

func StartTopUpConsumer(js nats.JetStreamContext, processor TopUpProcessor) (*nats.Subscription, error) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     topupStream,
		Subjects: []string{topupSubject},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return nil, err
	}

	sub, err := js.Subscribe(topupSubject, func(msg *nats.Msg) {
		var evt PendingTopUp
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			log.Printf("[NATS] balance.topup.pending unmarshal error: %v", err)
			_ = msg.Term()
			return
		}
		log.Printf("[NATS] processing queued top-up: target=%s amount=%.2f (queued %s)",
			evt.TargetUserID, evt.Amount, evt.QueuedAt)
		if err := processor.ProcessQueuedTopUp(evt); err != nil {
			log.Printf("[NATS] queued top-up failed for user %s: %v", evt.TargetUserID, err)
			_ = msg.NakWithDelay(5 * time.Second)
			return
		}
		log.Printf("[NATS] queued top-up completed for user %s: +%.2f", evt.TargetUserID, evt.Amount)
		_ = msg.Ack()
	}, nats.Durable(topupDurable), nats.ManualAck())
	if err != nil {
		return nil, err
	}
	log.Println("[NATS] auth-service subscribed to JetStream: balance.topup.pending")
	return sub, nil
}
