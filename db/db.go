package db

import "github.com/VictoriaMetrics/fastcache"

type StateMachine interface {
	Set([]byte, []byte)
	Get([]byte) []byte
}

type db struct {
	c *fastcache.Cache
}

func (d *db) Set(key []byte, value []byte) {
	d.c.Set(key, value)
}

func (d *db) Get(key []byte) []byte {
	return d.c.Get(nil, key)
}

func NewStateMachine() StateMachine {
	c := fastcache.New(0)
	return &db{
		c: c,
	}
}
