package checkout

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	streamName = "CHECKOUT"
	subject    = "checkout.pending"
)

type Queue struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewQueue(natsURL string) (*Queue, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		nc.Close()
		return nil, err
	}
	return &Queue{nc: nc, js: js}, nil
}

func (q *Queue) Close() {
	if q.nc != nil {
		q.nc.Close()
	}
}

type PendingCheckout struct {
	UserID   string `json:"userId"`
	QueuedAt string `json:"queuedAt"`
}

func (q *Queue) Enqueue(userID string) error {
	payload, err := json.Marshal(PendingCheckout{
		UserID:   userID,
		QueuedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	_, err = q.js.Publish(subject, payload, nats.MsgId("checkout-"+userID))
	return err
}
