package application

import (
	"sync"
)

type Coordinator struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewCoordinator(size int) *Coordinator {
	if size < 8 {
		size = 64
	}
	return &Coordinator{locks: make(map[string]*sync.Mutex, size)}
}
func (c *Coordinator) lock(key string) func() {
	c.mu.Lock()
	m := c.locks[key]
	if m == nil {
		m = &sync.Mutex{}
		c.locks[key] = m
	}
	c.mu.Unlock()

	m.Lock()
	c.mu.Lock()
	delete(c.locks, key)
	c.mu.Unlock()
	return m.Unlock
}
