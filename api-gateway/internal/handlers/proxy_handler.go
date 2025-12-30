package handlers

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/0Bleak/api-gateway/internal/proxy"
)

type ProxyHandler struct {
	loadBalancer *proxy.LoadBalancer
}

func NewProxyHandler(lb *proxy.LoadBalancer) *ProxyHandler {
	return &ProxyHandler{
		loadBalancer: lb,
	}
}

func (h *ProxyHandler) ProxyToService(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceURL, err := h.loadBalancer.GetServiceURL(serviceName)
		if err != nil {
			log.Printf("Failed to get service URL: %v", err)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api")
		targetURL := serviceURL + path
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			log.Printf("Failed to create proxy request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("Failed to proxy request: %v", err)
			http.Error(w, "Service unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
