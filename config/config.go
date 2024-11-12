package config

import (
	"errors"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

type Node struct {
	Id      int    `yaml:"id"`
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

func (n *Node) GetAddress() string {
	return net.JoinHostPort(n.Address, n.Port)
}

type Config struct {
	Nodes []Node
}

func (c *Config) GetNode(id int) (Node, error) {
	for _, n := range c.Nodes {
		if n.Id == id {
			return n, nil
		}
	}
	return Node{}, errors.New("config not found")
}

func ReadConfig(file string) (*Config, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var c Config
	err = yaml.Unmarshal(raw, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
