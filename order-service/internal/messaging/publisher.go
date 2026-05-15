package messaging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	orderStream        = "ORDERS"
	orderCreatedSubject = "order.created"
)

type Publisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewPublisher(nc *nats.Conn) *Publisher {
	p := &Publisher{nc: nc}
	if nc == nil {
		return p
	}
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("[NATS] JetStream unavailable for publisher: %v", err)
		return p
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     orderStream,
		Subjects: []string{orderCreatedSubject},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Printf("[NATS] ORDERS stream: %v", err)
		return p
	}
	p.js = js
	return p
}

type OrderEvent struct {
	OrderID     string  `json:"orderId"`
	UserID      string  `json:"userId"`
	TotalAmount float64 `json:"totalAmount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	EventType   string  `json:"eventType"`
}

func (p *Publisher) PublishOrderCreated(event OrderEvent) {
	if p.js != nil {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("[NATS] marshal error for %s: %v", orderCreatedSubject, err)
			return
		}
		if _, err := p.js.Publish(orderCreatedSubject, data, nats.MsgId("order-"+event.OrderID)); err != nil {
			log.Printf("[NATS] JetStream publish error for %s: %v", orderCreatedSubject, err)
			return
		}
		log.Printf("[NATS] published to %s (JetStream): %s", orderCreatedSubject, string(data))
		return
	}
	p.publish(orderCreatedSubject, event)
}

func (p *Publisher) PublishOrderConfirmed(event OrderEvent) {
	p.publish("order.confirmed", event)
}

func (p *Publisher) PublishOrderCancelled(event OrderEvent) {
	p.publish("order.cancelled", event)
}

func (p *Publisher) publish(subject string, payload interface{}) {
	if p.nc == nil {
		log.Printf("[NATS] connection is nil, skipping publish to %s", subject)
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[NATS] marshal error for %s: %v", subject, err)
		return
	}
	if err := p.nc.Publish(subject, data); err != nil {
		log.Printf("[NATS] publish error for %s: %v", subject, err)
		return
	}
	log.Printf("[NATS] published to %s: %s", subject, string(data))
}
