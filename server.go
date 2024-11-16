package server

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"sync/atomic"

	rpcx "github.com/smallnest/rpcx/server"

	"github.com/raft/config"
)

type Server struct {
	mu sync.Mutex

	id int

	state *state

	commitIndex atomic.Uint64 // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied atomic.Uint64 // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	// on leader
	nextIndex  []uint64 // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []uint64 // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	rpc *rpcx.Server

	config struct {
		*config.Config
		node *config.Node
	}

	exitChan chan struct{}
}

func (s *Server) logState() error {
	path := s.getUnderlyingFilePath()
	if path == "" {
		return errors.New("directory not specified in config")
	}
	if err := s.state.write(); err != nil {
		return err
	}
	return nil
}

func (s *Server) startRPCServer() error {
	rpcServer := rpcx.NewServer()
	if err := rpcServer.Register(s, ""); err != nil {
		return err
	}
	s.rpc = rpcServer
	go rpcServer.Serve("tcp", s.config.node.GetAddress())
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

func (s *Server) Listen() {
	for {
		select {
		case <-s.exitChan:
			return
		}
	}
}

func (s *Server) Close() error {
	s.exitChan <- struct{}{}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.logState(); err != nil {
		return err
	}
	if err := s.rpc.Close(); err != nil {
		return err
	}
	return nil
}

// try to read config file from disk
func (s *Server) initInternal() error {
	if s.config.Dir == "" {
		return errors.New("directory not specified in config")
	}
	if _, err := os.Stat(s.config.Dir); os.IsNotExist(err) {
		// newly created file for server
		err := s.ensureDir(s.config.Dir)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	st, err := newState(s.getUnderlyingFilePath())
	if err != nil {
		return err
	}
	s.state = st
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
		return fmt.Sprintf("%v", s.id)
	}()
	if s.config.Dir == "" {
		return ""
	}
	return path.Join(s.config.Dir, name)
}
