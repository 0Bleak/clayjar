package discovery

import (
	"fmt"
	"os"

	"github.com/hashicorp/consul/api"
)

type ConsulClient struct {
	client *api.Client
}

func NewConsulClient(address string) (*ConsulClient, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &ConsulClient{client: client}, nil
}

func (c *ConsulClient) RegisterService(serviceID, serviceName, port string) error {
	// Get the container hostname for health checks
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = serviceName
	}

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: hostname,
		Port:    parsePort(port),
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%s/health", hostname, port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	return c.client.Agent().ServiceRegister(registration)
}

func (c *ConsulClient) DeregisterService(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}

func parsePort(port string) int {
	var p int
	fmt.Sscanf(port, "%d", &p)
	return p
}
