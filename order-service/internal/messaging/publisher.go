package messaging

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	nc *nats.Conn
}

func NewPublisher(nc *nats.Conn) *Publisher {
	return &Publisher{nc: nc}
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
	p.publish("order.created", event)
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
