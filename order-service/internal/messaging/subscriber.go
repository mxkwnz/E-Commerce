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

type OrderStatusUpdater interface {
	ConfirmOrder(orderID string) error
	CancelOrder(orderID string) error
}

type Subscriber struct {
	nc      *nats.Conn
	updater OrderStatusUpdater
	subs    []*nats.Subscription
}

func NewSubscriber(nc *nats.Conn, updater OrderStatusUpdater) *Subscriber {
	return &Subscriber{nc: nc, updater: updater}
}

func (s *Subscriber) Subscribe() error {
	sub1, err := s.nc.Subscribe("payment.completed", s.handlePaymentCompleted)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub1)

	sub2, err := s.nc.Subscribe("payment.failed", s.handlePaymentFailed)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub2)

	log.Println("[NATS] order-service subscribed to: payment.completed, payment.failed")
	return nil
}

func (s *Subscriber) handlePaymentCompleted(msg *nats.Msg) {
	var event PaymentEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] payment.completed unmarshal error: %v", err)
		return
	}
	log.Printf("[NATS] payment.completed received for order %s", event.OrderID)
	if err := s.updater.ConfirmOrder(event.OrderID); err != nil {
		log.Printf("[NATS] failed to confirm order %s: %v", event.OrderID, err)
	}
}

func (s *Subscriber) handlePaymentFailed(msg *nats.Msg) {
	var event PaymentEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] payment.failed unmarshal error: %v", err)
		return
	}
	log.Printf("[NATS] payment.failed received for order %s", event.OrderID)
	if err := s.updater.CancelOrder(event.OrderID); err != nil {
		log.Printf("[NATS] failed to cancel order %s: %v", event.OrderID, err)
	}
}

func (s *Subscriber) Drain() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
