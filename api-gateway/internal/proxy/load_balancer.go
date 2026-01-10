package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/0Bleak/api-gateway/internal/discovery"
	"github.com/hashicorp/consul/api"
)

type LoadBalancer struct {
	consul      *discovery.ConsulClient
	mu          sync.Mutex
	counter     map[string]uint64
	cache       map[string][]*api.ServiceEntry
	cacheExpiry map[string]time.Time
	cacheTTL    time.Duration
}

func NewLoadBalancer(consul *discovery.ConsulClient) *LoadBalancer {
	lb := &LoadBalancer{
		consul:      consul,
		counter:     make(map[string]uint64),
		cache:       make(map[string][]*api.ServiceEntry),
		cacheExpiry: make(map[string]time.Time),
		cacheTTL:    5 * time.Second,
	}

	go lb.backgroundRefresh()
	return lb
}

func (lb *LoadBalancer) GetServiceURL(serviceName string) (string, error) {
	services, err := lb.getCachedInstances(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to get service instances: %w", err)
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	instance := lb.selectInstance(serviceName, services)
	return fmt.Sprintf("http://%s:%d", instance.Service.Address, instance.Service.Port), nil
}

func (lb *LoadBalancer) getCachedInstances(serviceName string) ([]*api.ServiceEntry, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if expiry, ok := lb.cacheExpiry[serviceName]; ok && time.Now().Before(expiry) {
		return lb.cache[serviceName], nil
	}

	services, err := lb.consul.GetServiceInstances(serviceName)
	if err != nil {
		return nil, err
	}

	lb.cache[serviceName] = services
	lb.cacheExpiry[serviceName] = time.Now().Add(lb.cacheTTL)

	return services, nil
}

func (lb *LoadBalancer) selectInstance(serviceName string, services []*api.ServiceEntry) *api.ServiceEntry {
	count := lb.counter[serviceName]
	index := count % uint64(len(services))
	lb.counter[serviceName] = count + 1

	return services[index]
}

func (lb *LoadBalancer) backgroundRefresh() {
	ticker := time.NewTicker(lb.cacheTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		lb.mu.Lock()
		serviceNames := make([]string, 0, len(lb.cache))
		for name := range lb.cache {
			serviceNames = append(serviceNames, name)
		}
		lb.mu.Unlock()

		for _, name := range serviceNames {
			services, err := lb.consul.GetServiceInstances(name)
			if err == nil {
				lb.mu.Lock()
				lb.cache[name] = services
				lb.cacheExpiry[name] = time.Now().Add(lb.cacheTTL)
				lb.mu.Unlock()
			}
		}
	}
}
