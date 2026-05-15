package messaging

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type PaymentEvent struct {
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
	EventType string  `json:"eventType"`
}

type UserBalanceProcessor interface {
	DeductBalance(userID string, amount float64) error
}

type Subscriber struct {
	nc        *nats.Conn
	processor UserBalanceProcessor
	subs      []*nats.Subscription
}

func NewSubscriber(nc *nats.Conn, processor UserBalanceProcessor) *Subscriber {
	return &Subscriber{nc: nc, processor: processor}
}

func (s *Subscriber) Subscribe() error {
	sub, err := s.nc.Subscribe("payment.completed", s.handlePaymentCompleted)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub)
	log.Println("[NATS] auth-service subscribed to: payment.completed")
	return nil
}

func (s *Subscriber) handlePaymentCompleted(msg *nats.Msg) {
	var event PaymentEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] payment.completed unmarshal error: %v", err)
		return
	}

	log.Printf("[NATS] payment.completed received: user=%s amount=%.2f", event.UserID, event.Amount)

	if err := s.processor.DeductBalance(event.UserID, event.Amount); err != nil {
		log.Printf("[NATS] balance deduction failed for user %s: %v", event.UserID, err)
		s.publishPaymentFailed(event, err.Error())
	} else {
		log.Printf("[NATS] balance deducted for user %s: %.2f", event.UserID, event.Amount)
	}
}

func (s *Subscriber) publishPaymentFailed(event PaymentEvent, reason string) {
	event.Status = "failed"
	event.EventType = "payment.failed"
	payload, _ := json.Marshal(event)
	_ = s.nc.Publish("payment.failed", payload)
}

func (s *Subscriber) Drain() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
