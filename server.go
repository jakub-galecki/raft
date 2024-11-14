package server

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"

	rpcx "github.com/smallnest/rpcx/server"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/raft/config"
	"github.com/raft/model"
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
}

type state struct {
	CurrentTerm atomic.Uint64 // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    atomic.Uint32 // candidateId that received vote in current term (or null if none)

	// log should be accessed with mutex locked
	Log []model.Entry

	w *os.File // underlying file
}

func (s *Server) serializeState() ([]byte, error) {
	return msgpack.Marshal(s.state)
}

func (s *Server) logState() error {
	path := s.getUnderlyingFilePath()
	if path == "" {
		return errors.New("directory not specified in config")
	}
	raw, err := s.serializeState()
	if err != nil {
		return err
	}
	err = s.state.w.Truncate(0)
	if err != nil {
		return err
	}
	s.state.w.Seek(0, io.SeekStart)
	_, err = s.state.w.Write(raw)
	if err != nil {
		return err
	}
	return nil
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
		err := s.ensureDir(path)
		if err != nil {
			return err
		}
		f, err := os.OpenFile("", os.O_CREATE|os.O_RDWR|os.O_SYNC, os.ModePerm)
		if err != nil {
			return err
		}
		s.state.w = f
		return nil
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
		return fmt.Sprintf("%v", s.id)
	}()
	if s.config.Dir == "" {
		return ""
	}
	return path.Join(s.config.Dir, name)
}
