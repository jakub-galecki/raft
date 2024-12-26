package model

type RequestVote struct {
	Term         uint64 // candidate’s term
	CandidateId  int    //candidate requesting vote
	LastLogIndex int    // index of candidate’s last log entry
	LastLogTerm  int    // term of candidate’s last log entry
}

type RequestVoteResponse struct {
	Term        uint64 // currentTerm, for candidate to update itself
	VoteGranted bool   // true means candidate received vote
}

/*
	Receiver implementation:
	1. Reply false if term < currentTerm (§5.1)
	2. If votedFor is null or candidateId, and candidate’s log is at
	least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
*/
