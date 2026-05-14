package messaging

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)


type OrderEvent struct {
	OrderID     string  `json:"orderId"`
	UserID      string  `json:"userId"`
	TotalAmount float64 `json:"totalAmount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	EventType   string  `json:"eventType"`
}


type PaymentEvent struct {
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
	EventType string  `json:"eventType"`
}

type PaymentProcessor interface {
	ProcessPaymentForOrder(orderID, userID string, amount float64, currency string) error
}

type Subscriber struct {
	nc        *nats.Conn
	processor PaymentProcessor
	subs      []*nats.Subscription
}

func NewSubscriber(nc *nats.Conn, processor PaymentProcessor) *Subscriber {
	return &Subscriber{nc: nc, processor: processor}
}

func (s *Subscriber) Subscribe() error {
	sub, err := s.nc.Subscribe("order.created", s.handleOrderCreated)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub)
	log.Println("[NATS] payment-service subscribed to: order.created")
	return nil
}

func (s *Subscriber) handleOrderCreated(msg *nats.Msg) {
	var event OrderEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] order.created unmarshal error: %v", err)
		return
	}
	log.Printf("[NATS] order.created received: order=%s user=%s amount=%.2f",
		event.OrderID, event.UserID, event.TotalAmount)

	
	if err := s.processor.ProcessPaymentForOrder(
		event.OrderID, event.UserID, event.TotalAmount, event.Currency,
	); err != nil {
		log.Printf("[NATS] payment processing failed for order %s: %v", event.OrderID, err)
		s.publishPaymentFailed(event)
		return
	}
	s.publishPaymentCompleted(event)
}

func (s *Subscriber) publishPaymentCompleted(event OrderEvent) {
	payload, _ := json.Marshal(PaymentEvent{
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		Amount:    event.TotalAmount,
		Currency:  event.Currency,
		Status:    "paid",
		EventType: "payment.completed",
	})
	_ = s.nc.Publish("payment.completed", payload)
	log.Printf("[NATS] published payment.completed for order %s", event.OrderID)
}

func (s *Subscriber) publishPaymentFailed(event OrderEvent) {
	payload, _ := json.Marshal(PaymentEvent{
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		Amount:    event.TotalAmount,
		Currency:  event.Currency,
		Status:    "failed",
		EventType: "payment.failed",
	})
	_ = s.nc.Publish("payment.failed", payload)
	log.Printf("[NATS] published payment.failed for order %s", event.OrderID)
}

func (s *Subscriber) Drain() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
