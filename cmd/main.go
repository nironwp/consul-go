package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

type Service struct {
	consulClient *api.Client
}

const (
	PORT    = 3000
	ttl     = time.Second * 10
	checkID = "checkalive"
)

func NewService() *Service {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		log.Fatal(err)
	}
	return &Service{
		consulClient: client,
	}
}

func (s *Service) updateHealthCheck() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		err := s.consulClient.Agent().UpdateTTL(checkID, "online", api.HealthPassing)
		if err != nil {
			log.Fatal(err)
		}
		<-ticker.C
	}
}

func (s *Service) Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		log.Fatal(err)
	}
	s.registerService()

	go s.updateHealthCheck()
	println("Server listening on port", PORT)
	s.acceptLoop(ln)
}

func (s *Service) acceptLoop(ln net.Listener) {
	for {
		_, err := ln.Accept()

		if err != nil {
			log.Fatal(err)
		}
	}
}

func (s *Service) registerService() {
	check := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: ttl.String(),
		TLSSkipVerify:                  true,
		TTL:                            ttl.String(),
		CheckID:                        checkID,
	}
	reg := &api.AgentServiceRegistration{
		ID:      "login_service",
		Name:    "mycluster",
		Tags:    []string{"login"},
		Address: "127.0.0.1",
		Port:    3000,
		Check:   check,
	}

	query := map[string]any{
		"type":        "service",
		"service":     "mycluster",
		"passingonly": true,
	}
	plan, err := watch.Parse(query)

	if err != nil {
		log.Fatal(err)
	}

	plan.HybridHandler = func(index watch.BlockingParamVal, result any) {
		switch msg := result.(type) {
		case []*api.ServiceEntry:
			for _, entry := range msg {
				fmt.Println(
					"new member joined",
					entry.Service,
				)

				fmt.Println("Service primary infos",
					entry.Service.Address,
					entry.Service.Port,
					entry.Service.ID,
				)

				fmt.Println("Service secondary infos",
					entry.Service.Service,
					entry.Service.Proxy,
					entry.Service.Connect,
					entry.Service.Namespace,
				)

				fmt.Println("Service meta infos",
					entry.Service.Tags,
					entry.Service.Meta,
				)

				fmt.Println("Service check infos",
					entry.Service.Weights,
					entry.Service.EnableTagOverride,
					entry.Service.Datacenter)
			}
		}

		fmt.Println("update cluster", result)
	}

	go func() {
		plan.RunWithConfig("", &api.Config{})
	}()

	if err := s.consulClient.Agent().ServiceRegister(reg); err != nil {
		log.Fatal(err)
	}
}

func main() {
	s := NewService()

	s.Start()
}
