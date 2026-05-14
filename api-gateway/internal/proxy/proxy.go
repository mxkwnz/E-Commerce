package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"api-gateway/internal/config"
)

type Proxy struct {
	auth    *httputil.ReverseProxy
	product *httputil.ReverseProxy
	order   *httputil.ReverseProxy
	payment *httputil.ReverseProxy
}

func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		auth:    newReverseProxy(cfg.AuthServiceURL, "auth"),
		product: newReverseProxy(cfg.ProductServiceURL, "product"),
		order:   newReverseProxy(cfg.OrderServiceURL, "order"),
		payment: newReverseProxy(cfg.PaymentServiceURL, "payment"),
	}
}

func (p *Proxy) Auth(w http.ResponseWriter, r *http.Request) {
	p.auth.ServeHTTP(w, r)
}

func (p *Proxy) Product(w http.ResponseWriter, r *http.Request) {
	p.product.ServeHTTP(w, r)
}

func (p *Proxy) Order(w http.ResponseWriter, r *http.Request) {
	p.order.ServeHTTP(w, r)
}

func (p *Proxy) Payment(w http.ResponseWriter, r *http.Request) {
	p.payment.ServeHTTP(w, r)
}

func newReverseProxy(target, name string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("[PROXY] invalid URL for %s: %v", name, err)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	original := rp.Director
	rp.Director = func(req *http.Request) {
		original(req)
		req.Host = u.Host
		if req.Header.Get("X-Forwarded-For") == "" {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Gateway", "api-gateway")
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[PROXY] %s upstream error: %v", name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"upstream service unavailable"}`))
	}

	rp.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Served-By", "api-gateway")
		return nil
	}

	log.Printf("[PROXY] registered %s → %s", name, target)
	return rp
}
