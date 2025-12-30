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

func (c *ConsulClient) GetServiceInstances(serviceName string) ([]*api.ServiceEntry, error) {
	services, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	return services, nil
}

func (c *ConsulClient) RegisterService(service *api.AgentServiceRegistration) error {
	return c.client.Agent().ServiceRegister(service)
}

func (c *ConsulClient) DeregisterService(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}
