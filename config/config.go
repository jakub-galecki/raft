package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Node struct {
	Id      int    `yaml:"id"`
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

type Config struct {
	Nodes []Node
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
