package server

import (
	rpcx "github.com/smallnest/rpcx/server"

	"github.com/raft/config"
)

type Server struct {
	config struct {
		*config.Config
		node *config.Node
	}

	rpc *rpcx.Server
}

func (s *Server) startRPCServer() error {
	rpcServer := rpcx.NewServer()
	rpcServer.Register(s, "")
	err := rpcServer.Serve("tcp", s.config.node.GetAddress())
	if err != nil {
		return err
	}
	s.rpc = rpcServer
	return nil
}

func NewServer(id int, conf *config.Config) (*Server, error) {
	node, err := conf.GetNode(id)
	if err != nil {
		return nil, err
	}
	s := &Server{}
	s.config.Config = conf
	s.config.node = &node
	if err := s.startRPCServer(); err != nil {
		return nil, err
	}
	return s, nil
}
