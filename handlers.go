package server

import (
	"context"
	"log/slog"

	"github.com/raft/model"
)

func (s *Server) AppendEntries(ctx context.Context, req model.AppendEntriesRequest, res *model.AppendEntriesResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	term := s.state.getCurrentTerm()

	if req.Term < term {
		res.Success = false
		res.Term = term
		return nil
	}

	return nil
}

func (s *Server) Vote(ctx context.Context, req model.RequestVote, res *model.RequestVoteResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	term := s.state.getCurrentTerm()
	if req.Term < term {
		res.Term = s.state.getCurrentTerm()
		res.VoteGranted = false
		return nil
	}

	var (
		votedFor         = s.state.VotedFor.Load()
		lastLogIndex     = s.getLastLogIndex()
		lastLogTerm      = s.getLogTermAt(lastLogIndex)
		votedForMatch    = (votedFor == -1 || votedFor == int32(req.CandidateId))
		logMatch         = req.LastLogTerm > lastLogTerm || (req.LastLogTerm == lastLogTerm && req.LastLogIndex >= lastLogIndex)
		currentTermMatch = req.Term == term
		granted          = currentTermMatch && votedForMatch && logMatch
	)
	if granted {
		s.state.VotedFor.Store(int32(req.CandidateId))
		s.resetElectionTicker()
	}
	res.Term = term
	res.VoteGranted = granted
	if err := s.logState(); err != nil {
		s.l.Error("error while logging state", slog.Int("server", s.id), slog.Any("error", err.Error()))
		return err
	}
	return nil
}
