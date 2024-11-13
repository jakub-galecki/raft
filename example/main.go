package main

import (
	"fmt"

	server "github.com/raft"
	"github.com/raft/config"
)

func main() {
    conf, err := config.ReadConfig("../testdata/config.yaml")
    if err != nil {
        panic(err)
    }
    s, err := server.NewServer(1, conf)
    if err != nil {
        panic(err)
    }
    s.
}
