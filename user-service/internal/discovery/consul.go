package discovery

import (
	"fmt"

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
	registration := &api.AgentServiceRegistration{
		ID:   serviceID,
		Name: serviceName,
		Port: parsePort(port),
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%s/health", serviceID, port),
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
