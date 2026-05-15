package messaging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	checkoutStream  = "CHECKOUT"
	checkoutSubject = "checkout.pending"
	checkoutDurable = "order-checkout-worker"
)

type CheckoutProcessor interface {
	RunCheckout(userID string) error
}

type pendingCheckout struct {
	UserID   string `json:"userId"`
	QueuedAt string `json:"queuedAt"`
}

func StartCheckoutConsumer(js nats.JetStreamContext, processor CheckoutProcessor) (*nats.Subscription, error) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     checkoutStream,
		Subjects: []string{checkoutSubject},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return nil, err
	}

	sub, err := js.Subscribe(checkoutSubject, func(msg *nats.Msg) {
		var evt pendingCheckout
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			log.Printf("[NATS] checkout.pending unmarshal error: %v", err)
			_ = msg.Term()
			return
		}
		log.Printf("[NATS] processing queued checkout for user %s (queued %s)", evt.UserID, evt.QueuedAt)
		if err := processor.RunCheckout(evt.UserID); err != nil {
			log.Printf("[NATS] queued checkout failed for user %s: %v", evt.UserID, err)
			_ = msg.NakWithDelay(5 * time.Second)
			return
		}
		log.Printf("[NATS] queued checkout completed for user %s", evt.UserID)
		_ = msg.Ack()
	}, nats.Durable(checkoutDurable), nats.ManualAck())
	if err != nil {
		return nil, err
	}
	log.Println("[NATS] order-service subscribed to JetStream: checkout.pending")
	return sub, nil
}
