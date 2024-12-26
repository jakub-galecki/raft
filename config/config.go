package config

import (
	"errors"
	"net"
	"os"

	"gopkg.in/yaml.v3"

	rpcx "github.com/smallnest/rpcx/client"
)

// todo: close connection

type Node struct {
	Id      int    `yaml:"id"`
	Address string `yaml:"address"`
	Port    string `yaml:"port"`

	Conn rpcx.XClient
}

func (n *Node) Connect() error {
	addr := n.GetAddress()
	d, err := rpcx.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		return err
	}
	n.Conn = rpcx.NewXClient("", rpcx.Failover, rpcx.RandomSelect, d, rpcx.DefaultOption)
	return nil
}

func (n *Node) GetAddress() string {
	return net.JoinHostPort(n.Address, n.Port)
}

type Config struct {
	Dir   string `yaml:"dir"`
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
