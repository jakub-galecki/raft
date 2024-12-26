package server

import (
	"errors"
	"io"
	"os"
	"sync/atomic"

	"github.com/raft/model"
	"github.com/vmihailenco/msgpack/v5"
)

type state struct {
	CurrentTerm atomic.Uint64 // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	VotedFor    atomic.Int32  // candidateId that received vote in current term (or null if none)

	// log should be accessed with mutex locked
	Log []model.Entry

	file *os.File // underlying file
}

func (s *state) getCurrentTerm() uint64 {
	return s.CurrentTerm.Load()
}

func (s *state) setCurrentTerm(term uint64) {
	s.CurrentTerm.Store(term)
}

func (s *state) increaseCurrentTerm() uint64 {
	return s.CurrentTerm.Add(1)
}

func (s *state) getVotedFor() int {
	return int(s.VotedFor.Load())
}

func (s *state) setVotedFor(term int) {
	s.VotedFor.Store(int32(term))
}

func (s *state) increaseVotedFor() int32 {
	return s.VotedFor.Add(1)
}

func newState(file string) (*state, error) {
	st := new(state)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_SYNC, os.ModePerm)
	if err != nil {
		return nil, err
	}
	st.file = f
	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = msgpack.Unmarshal(raw, &st)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return st, nil
}

func (s *state) rewindFile() error {
	err := s.file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = s.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func (s *state) write() error {
	if err := s.rewindFile(); err != nil {
		return err
	}
	raw, err := s.serialize()
	if err != nil {
		return err
	}
	_, err = s.file.Write(raw)
	if err != nil {
		return err
	}
	return nil
}

func (s *state) serialize() ([]byte, error) {
	return msgpack.Marshal(s)
}
