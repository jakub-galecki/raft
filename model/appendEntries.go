package model

// Invoked by candidates to gather votes
type AppendEntriesRequest struct {
	Term         uint64  // leader’s term
	LeaderId     int     // so follower can redirect clients
	PrevLogIndex int     // index of log entry immediately preceding new ones
	PrevLogTerm  int     // term of prevLogIndex entry
	Entries      []Entry // log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int     //leader’s commitIndex
}

type AppendEntriesResponse struct {
	Term    uint64 // currentTerm, for leader to update itself
	Success bool   // true if follower contained entry matching prevLogIndex and prevLogTerm
}

/*
	Receiver implementation:
		1. Reply false if term < currentTerm (§5.1)
		2. Reply false if log doesn’t contain an entry at prevLogIndex
		whose term matches prevLogTerm (§5.3)
		3. If an existing entry conflicts with a new one (same index
		but different terms), delete the existing entry and all that
		follow it (§5.3)
		4. Append any new entries not already in the log
		5. If leaderCommit > commitIndex, set commitIndex =
		min(leaderCommit, index of last new entry)
*/
