package config

import "os"

type Config struct {
	Port              string
	AuthServiceURL    string
	ProductServiceURL string
	OrderServiceURL   string
	PaymentServiceURL string
}

func Load() *Config {
	return &Config{
		Port:              getenv("PORT", "8080"),
		AuthServiceURL:    getenv("AUTH_SERVICE_URL", "http://localhost:8081"),
		ProductServiceURL: getenv("PRODUCT_SERVICE_URL", "http://localhost:8082"),
		OrderServiceURL:   getenv("ORDER_SERVICE_URL", "http://localhost:8083"),
		PaymentServiceURL: getenv("PAYMENT_SERVICE_URL", "http://localhost:8084"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
