package main

import (
	"sync"
)

type threadSafeEnvironments struct {
	*sync.RWMutex
	envMap map[string]Environment
	ready  bool
}

func newThreadSafeEnvironments() *threadSafeEnvironments {
	return &threadSafeEnvironments{&sync.RWMutex{}, make(map[string]Environment), false}
}

func (tse *threadSafeEnvironments) len() int {
	tse.RLock()
	defer tse.RUnlock()
	return len(tse.envMap)
}

func (tse *threadSafeEnvironments) names() []string {
	tse.RLock()
	defer tse.RUnlock()
	var s []string
	for n := range tse.envMap {
		s = append(s, n)
	}
	return s
}

func (tse *threadSafeEnvironments) environment(name string) Environment {
	tse.RLock()
	defer tse.RUnlock()
	return tse.envMap[name]
}

func (tse *threadSafeEnvironments) areReady() bool {
	tse.RLock()
	defer tse.RUnlock()
	return tse.ready
}
