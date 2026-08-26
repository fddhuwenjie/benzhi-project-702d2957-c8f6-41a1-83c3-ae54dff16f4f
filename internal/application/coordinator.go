package application

import (
	"hash/fnv"
	"sync"
)

type Coordinator struct{ stripes []sync.Mutex }

func NewCoordinator(size int) *Coordinator {
	if size < 8 {
		size = 64
	}
	return &Coordinator{stripes: make([]sync.Mutex, size)}
}
func (c *Coordinator) lock(key string) func() {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	m := &c.stripes[int(h.Sum32())%len(c.stripes)]
	m.Lock()
	return m.Unlock
}
