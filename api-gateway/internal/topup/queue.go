package topup

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	streamName = "BALANCE"
	subject    = "balance.topup.pending"
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

type PendingTopUp struct {
	TargetUserID string  `json:"targetUserId"`
	AccessToken  string  `json:"accessToken"`
	ActorUserID  string  `json:"actorUserId,omitempty"`
	Amount       float64 `json:"amount"`
	QueuedAt     string  `json:"queuedAt"`
}

func (q *Queue) Enqueue(targetUserID, accessToken, actorUserID string, amount float64) error {
	payload, err := json.Marshal(PendingTopUp{
		TargetUserID: targetUserID,
		AccessToken:  accessToken,
		ActorUserID:  actorUserID,
		Amount:       amount,
		QueuedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	msgID := fmt.Sprintf("topup-%s-%d", targetUserID, time.Now().UnixNano())
	_, err = q.js.Publish(subject, payload, nats.MsgId(msgID))
	return err
}
