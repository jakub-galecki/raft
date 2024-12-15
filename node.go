package server

import "github.com/raft/config"

type nodes []*innerNode

func initNodes(conf *config.Config) nodes {
	res := make([]*innerNode, len(conf.Nodes))
	for _, c := range conf.Nodes {
		res = append(res, &innerNode{
			&c,
		})
	}
	return res
}

type innerNode struct {
	*config.Node
}

func (n *innerNode) Addr() string {
	return n.GetAddress()
}
