package messaging

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type OrderEvent struct {
	OrderID   string      `json:"orderId"`
	UserID    string      `json:"userId"`
	Items     []OrderItem `json:"items"`
	EventType string      `json:"eventType"`
}

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type StockManager interface {
	ReserveStock(productID string, amount int) error
	ReleaseStock(productID string, amount int) error
}

type Subscriber struct {
	nc    *nats.Conn
	stock StockManager
	subs  []*nats.Subscription
}

func NewSubscriber(nc *nats.Conn, stock StockManager) *Subscriber {
	return &Subscriber{nc: nc, stock: stock}
}

func (s *Subscriber) Subscribe() error {
	sub1, err := s.nc.Subscribe("order.confirmed", s.handleOrderConfirmed)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub1)

	sub2, err := s.nc.Subscribe("order.cancelled", s.handleOrderCancelled)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub2)

	log.Println("[NATS] product-service subscribed to: order.confirmed, order.cancelled")
	return nil
}

func (s *Subscriber) handleOrderConfirmed(msg *nats.Msg) {
	var event OrderEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] order.confirmed unmarshal error: %v", err)
		return
	}
	log.Printf("[NATS] order.confirmed: %s — releasing reservation", event.OrderID)
	for _, item := range event.Items {
		if err := s.stock.ReleaseStock(item.ProductID, item.Quantity); err != nil {
			log.Printf("[NATS] ReleaseStock failed for product %s: %v", item.ProductID, err)
		}
	}
}

func (s *Subscriber) handleOrderCancelled(msg *nats.Msg) {
	var event OrderEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[NATS] order.cancelled unmarshal error: %v", err)
		return
	}
	log.Printf("[NATS] order.cancelled: %s — releasing reserved stock", event.OrderID)
	for _, item := range event.Items {
		if err := s.stock.ReleaseStock(item.ProductID, item.Quantity); err != nil {
			log.Printf("[NATS] ReleaseStock failed for product %s: %v", item.ProductID, err)
		}
	}
}

func (s *Subscriber) Drain() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
