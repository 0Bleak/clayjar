package proxy

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/0Bleak/api-gateway/internal/discovery"
	"github.com/hashicorp/consul/api"
)

type LoadBalancer struct {
	consul *discovery.ConsulClient
	mu     sync.RWMutex
}

func NewLoadBalancer(consul *discovery.ConsulClient) *LoadBalancer {
	return &LoadBalancer{
		consul: consul,
	}
}

func (lb *LoadBalancer) GetServiceURL(serviceName string) (string, error) {
	services, err := lb.consul.GetServiceInstances(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to get service instances: %w", err)
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	instance := lb.selectInstance(services)
	return fmt.Sprintf("http://%s:%d", instance.Service.Address, instance.Service.Port), nil
}

func (lb *LoadBalancer) selectInstance(services []*api.ServiceEntry) *api.ServiceEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	index := rand.Intn(len(services))
	return services[index]
}
