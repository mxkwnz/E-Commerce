package messaging

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type AuthEvent struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	EventType string `json:"eventType"`
}

type Publisher struct {
	nc *nats.Conn
}

func NewPublisher(nc *nats.Conn) *Publisher {
	return &Publisher{nc: nc}
}

func (p *Publisher) PublishUserRegistered(event AuthEvent) {
	event.EventType = "user.registered"
	p.publish("user.registered", event)
}

func (p *Publisher) PublishUserDeleted(event AuthEvent) {
	event.EventType = "user.deleted"
	p.publish("user.deleted", event)
}

func (p *Publisher) PublishPasswordChanged(event AuthEvent) {
	event.EventType = "user.password_changed"
	p.publish("user.password_changed", event)
}

func (p *Publisher) publish(subject string, payload interface{}) {
	if p.nc == nil {
		log.Printf("[NATS] auth publisher: connection nil, skipping %s", subject)
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
	log.Printf("[NATS] published %s: %s", subject, string(data))
}
