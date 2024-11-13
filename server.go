package server

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	rpcx "github.com/smallnest/rpcx/server"

	"github.com/raft/config"
	"github.com/raft/model"
)

type Server struct {
    id int
	config struct {
		*config.Config
		node *config.Node
	}

    mu struct {
        sync.Mutex

        currentTerm int 
        votedFor int 
        log []model.Entry
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
    s := &Server{id: id}
	s.config.Config = conf
	s.config.node = &node
    if err := s.initInternal(); err != nil {
        return nil, err
    }
	if err := s.startRPCServer(); err != nil {
		return nil, err
	}
	return s, nil
}

// try to read config file from disk 
func (s *Server) initInternal() error {
    path := s.getUnderlyingFilePath()
    if path == "" {
        return errors.New("directory not specified in config")
    }
    
    if _, err := os.Stat(path); os.IsNotExist(err) {
        // newly created file for server 
        return s.ensureDir(path)
    } else if err != nil {
        return nil
    } 
    
    // read file
    
    return nil
}

func (s *Server) ensureDir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
        return err
	}
    return nil
}


func (s *Server) getUnderlyingFilePath() string {
    name := func() string {
        return fmt.Sprintf("%s", s.id)
    }()
    if s.config.Dir == "" {
        return ""
    }
    return path.Join(s.config.Dir, name)
}
