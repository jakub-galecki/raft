package model

type Entry struct {
	Term    int    // when entry was received by leader
	Command []byte // command for state machine
}
