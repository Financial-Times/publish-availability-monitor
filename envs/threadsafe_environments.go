package envs

import (
	"sync"
)

// Environment defines an environment in which the publish metrics should be checked
type Environment struct {
	Name     string `json:"name"`
	ReadURL  string `json:"read-url"`
	S3Url    string `json:"s3-url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ThreadSafeEnvironments struct {
	*sync.RWMutex
	EnvMap map[string]Environment
	ready  bool
}

func NewThreadSafeEnvironments() *ThreadSafeEnvironments {
	return &ThreadSafeEnvironments{&sync.RWMutex{}, make(map[string]Environment), false}
}

func (tse *ThreadSafeEnvironments) Len() int {
	tse.RLock()
	defer tse.RUnlock()
	return len(tse.EnvMap)
}

func (tse *ThreadSafeEnvironments) Names() []string {
	tse.RLock()
	defer tse.RUnlock()
	var s []string
	for n := range tse.EnvMap {
		s = append(s, n)
	}
	return s
}

func (tse *ThreadSafeEnvironments) Environment(name string) Environment {
	tse.RLock()
	defer tse.RUnlock()
	return tse.EnvMap[name]
}

func (tse *ThreadSafeEnvironments) AreReady() bool {
	tse.RLock()
	defer tse.RUnlock()
	return tse.ready
}
