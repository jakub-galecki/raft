package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	rpcx "github.com/smallnest/rpcx/server"

	"github.com/raft/config"
	"github.com/raft/model"
)

type ServerRole uint8

const (
	Leader ServerRole = iota
	Follower
	Candidate
)

func (sr ServerRole) String() string {
	switch sr {
	case Leader:
		return "leader"
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	}
	return ""
}

const (
	electionTickerDuration = 1 * time.Second
)

type Server struct {
	mu sync.Mutex

	id int

	state *state

	commitIndex atomic.Uint64 // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied atomic.Uint64 // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	// on leader
	nextIndex  []int // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []int // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	rpc *rpcx.Server

	config struct {
		*config.Config
		node *config.Node
	}

	nodes []*innerNode

	electionTicker *time.Ticker
	exitChan       chan struct{}
	role           ServerRole

	l *slog.Logger
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
	go func() {
		// todo: handle error
		err := rpcServer.Serve("tcp", s.config.node.GetAddress())
		if err != nil {
			panic(err)
		}
	}()
	return nil
}

func (s *Server) resetElectionTicker() {
	s.electionTicker.Reset(electionTickerDuration)
}

func NewServer(id int, conf *config.Config) (*Server, error) {
	node, err := conf.GetNode(id)
	if err != nil {
		return nil, err
	}
	s := &Server{id: id}
	s.config.Config = conf
	s.config.node = &node

	// todo: take level from some config...
	s.l = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	s.nodes = initNodes(conf)
	s.electionTicker = time.NewTicker(electionTickerDuration)
	s.role = Follower
	if err := s.initInternal(); err != nil {
		return nil, err
	}
	if err := s.startRPCServer(); err != nil {
		return nil, err
	}
	go s.startElectionWorker()
	return s, nil
}

func (s *Server) isLeader() bool {
	return s.role == Leader
}

func (s *Server) startElectionWorker() {
	for range s.electionTicker.C {
		slog.Debug("Election ticker")

		if s.isLeader() {
			// no point to start election from leader
			s.resetElectionTicker()
			break
		}

		func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			err := s.elect()
			if err != nil {
				s.l.Error("error during election")
			}
		}()
	}
}

func (s *Server) getLastLogIndex() int {
	return len(s.state.Log) - 1
}

func (s *Server) getLogTermAt(index int) int {
	if index > len(s.state.Log) {
		return -1
	}
	return s.state.Log[index].Term
}

func (s *Server) appendEntriesLeader() {
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.role != Leader {
			s.l.Warn("appendEntries invoked on non leader", slog.String("role", s.role.String()))
			return
		}
	}()

	getEntiresFromIndex := func(i int) []model.Entry {
		if i < len(s.state.Log)-1 {
			return []model.Entry{s.state.Log[i]}
		}
		return []model.Entry{}
	}

	term := s.state.getCurrentTerm()
	for i := range s.nodes {
		n := s.nodes[i]
		go func() {
			s.mu.Lock()
			var (
				nextIndex, prevIndex = s.nextIndex[i], s.nextIndex[i] - 1
				prevTerm             = s.state.Log[prevIndex].Term
				ents                 = getEntiresFromIndex(nextIndex)
			)
			req := model.AppendEntriesRequest{
				Term:         term,
				LeaderId:     s.id,
				PrevLogIndex: prevIndex,
				PrevLogTerm:  prevTerm,
				Entries:      ents,
				LeaderCommit: s.commitIndex.Load(),
			}
			s.mu.Unlock()
			var response model.AppendEntriesResponse
			err := n.Conn.Call(context.Background(), "AppendEntries", req, &response)
			if err != nil {
				s.l.Error("error during AppendEntries", slog.String("err", err.Error()))
				return
			}
			s.mu.Lock()
			if response.Term > term {
				s.l.Debug("Received bigger term in append entries response. Transition to follower",
					slog.Int("server", s.id),
					slog.Uint64("current_term", term),
					slog.Uint64("received_term", response.Term))
				s.transistionToFollower(response.Term)
				s.mu.Unlock()
				return
			}

			if response.Success {
				var (
					newNextIndex = nextIndex + len(ents)
					commitIndex  = s.commitIndex.Load()
				)
				s.nextIndex[i] = newNextIndex
				s.matchIndex[i] = newNextIndex - 1

				for j := commitIndex + 1; j < uint64(len(s.state.Log)); j++ {
					if uint64(s.state.Log[j].Term) == s.state.getCurrentTerm() {
						if s.reachedLogQuorum(j) {
							s.commitIndex.Store(j)
						}
					}
				}

				if s.commitIndex.Load() > commitIndex {
					// todo: trigger applying to state machine
				}
			}
			s.mu.Unlock()
		}()
	}
}

func (s *Server) reachedLogQuorum(i uint64) bool {
	enoughVotes := func(c int) bool {
		if c*2 >= len(s.nodes) {
			return true
		}
		return false
	}

	nodesCount := 1
	for j := range s.nodes {
		if uint64(s.matchIndex[j]) >= i {
			nodesCount++
		}

		if enoughVotes(nodesCount) {
			return true
		}
	}
	return false
}

// elect is responsible for starting new election where caller
// is candidate. s.mu must be held by the caller
func (s *Server) elect() error {
	s.resetElectionTicker()

	var (
		lastLogIndex = s.getLastLogIndex()
		lastLogTerm  = s.getLogTermAt(lastLogIndex)
		term         = s.state.CurrentTerm.Add(1)
		total        = 1
	)

	s.role = Candidate
	s.state.VotedFor.Store(int32(s.id))

	s.l.Debug("starting election", slog.Int("server", s.id))

	req := model.RequestVote{
		Term:         term,
		CandidateId:  s.id,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	// todo: run in parallel
	for _, n := range s.nodes {
		var (
			reply       model.RequestVoteResponse
			ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
		)
		defer cancel()
		err := n.Conn.Call(ctx, "Vote", req, &reply)
		if err != nil {
			s.l.Error("error while calling node", slog.String("node", n.GetAddress()), slog.String("err", err.Error()))
			// throwing here error would be bad as we want to proceed with raft
			// even with some nodes are down. Just continue to the next node
			continue
		}

		if s.role != Candidate {
			s.l.Debug("role changed during election while waiting for reponse", slog.Int("server", s.id))
			return nil
		}

		if uint64(reply.Term) < term {
			s.l.Debug("rejecting rpc reposne as term is smaller", slog.Int("server", s.id),
				slog.Uint64("received_term", reply.Term),
				slog.Uint64("term", term))
		} else if uint64(reply.Term) > term {
			// todo: transition to follower role
			s.transistionToFollower(uint64(reply.Term))
		} else { // reply.Term == term
			if !reply.VoteGranted {
				continue
			}
			total += 1
			if total*2 > len(s.nodes)+1 {
				s.l.Debug("election success", slog.Int("server", s.id))
				s.transitionToLeader()
				return nil
			}
		}

	}

	return nil
}

func (s *Server) transitionToLeader() {
	s.role = Leader
	for i := range s.nodes {
		s.nextIndex[i] = len(s.state.Log)
		s.matchIndex[i] = -1
	}
	go s.startSendingHearbeats(time.NewTicker(50 * time.Millisecond))
}

func (s *Server) transistionToFollower(term uint64) {
	s.resetElectionTicker()
	s.role = Follower
	s.state.CurrentTerm.Store(term)
	s.state.VotedFor.Store(-1)
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

func (s *Server) appendEntries() {}

func (s *Server) startSendingHearbeats(c *time.Ticker) {
	s.appendEntries()

	for range c.C {
		s.appendEntries()
	}
}
