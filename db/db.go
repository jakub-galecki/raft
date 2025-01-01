package db

import "github.com/VictoriaMetrics/fastcache"

type StateMachine interface {
	Apply([]byte) ([]byte, error)
}

type db struct {
	c *fastcache.Cache
}

func (d *db) set(key []byte, value []byte) {
	d.c.Set(key, value)
}

func (d *db) get(key []byte) []byte {
	return d.c.Get(nil, key)
}

func (d *db) Apply(cmd []byte) ([]byte, error) {
	return nil, nil
}

func NewStateMachine() StateMachine {
	c := fastcache.New(0)
	return &db{
		c: c,
	}
}
