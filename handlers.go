package server

import (
	"context"

	"github.com/raft/model"
)

func (s *Server) AppendEntries(ctx context.Context, req model.AppendEntriesRequest, res *model.AppendEntriesResponse) error {
	return nil
}

func (s *Server) Vote(ctx context.Context, req model.RequestVote, res *model.RequestVoteResponse) error {
	return nil
}
