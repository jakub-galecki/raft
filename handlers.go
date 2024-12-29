package server

import (
	"context"
	"log/slog"

	"github.com/raft/model"
)

func (s *Server) appendInternal(from int, entires []model.Entry) bool {
	getAt := func(ents []model.Entry, i int) (model.Entry, bool) {
		if i > len(ents) {
			return model.Entry{}, false
		}
		return ents[i], true
	}

	matchTerm := func(i int, e *model.Entry) bool {
		logEntry, found := getAt(s.state.Log, i)
		if !found {
			return false
		}
		return logEntry.Term == e.Term
	}

	entriesCounter := 0
	for i := from; i < from+len(entires); i++ {
		e, found := getAt(entires, entriesCounter)
		if !found {
			s.l.Warn("index not found in entries", slog.String("method", "findConflicting"), slog.Int("index", entriesCounter))
			return false
		}
		if !matchTerm(i, &e) {
			// remove conflicting entry
			s.state.Log = s.state.Log[:i]
		}
		s.state.Log = append(s.state.Log, e)
		entriesCounter += 1
	}
	return true
}

func (s *Server) AppendEntries(ctx context.Context, req model.AppendEntriesRequest, res *model.AppendEntriesResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	term := s.state.getCurrentTerm()
	s.resetElectionTicker()

	switch {
	case req.Term < term:
		res.Success = false
		res.Term = term
		return nil
	case req.Term > term:
		s.transistionToFollower(req.Term)
		s.logState()
	}

	if req.Term == term {
		prevLogMatch := req.PrevLogIndex < len(s.state.Log) && req.PrevLogTerm == s.state.Log[req.PrevLogIndex].Term
		if req.PrevLogIndex == -1 || prevLogMatch {
			res.Success = true
			if s.appendInternal(req.PrevLogIndex+1, req.Entries) {
				if req.LeaderCommit > s.commitIndex.Load() {
                    s.commitIndex.Store(min(req.LeaderCommit, uint64(len(s.state.Log)-1)))
				}
                if err := s.logState(); err != nil {
                    return err
                }
			}
		}
	}
	res.Term = term
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
		votedForMatch    = votedFor == -1 || votedFor == int32(req.CandidateId)
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
