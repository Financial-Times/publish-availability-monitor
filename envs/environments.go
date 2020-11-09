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

// Environments provides a thread-safe collection of Environment structs
type Environments struct {
	*sync.RWMutex
	EnvMap map[string]Environment
	ready  bool
}

func NewEnvironments() *Environments {
	return &Environments{&sync.RWMutex{}, make(map[string]Environment), false}
}

func (tse *Environments) Len() int {
	tse.RLock()
	defer tse.RUnlock()
	return len(tse.EnvMap)
}

func (tse *Environments) Names() []string {
	tse.RLock()
	defer tse.RUnlock()

	names := make([]string, tse.Len())

	i := 0
	for name := range tse.EnvMap {
		names[i] = name
		i++
	}

	return names
}

func (tse *Environments) Environment(name string) Environment {
	tse.RLock()
	defer tse.RUnlock()
	return tse.EnvMap[name]
}

func (tse *Environments) AreReady() bool {
	tse.RLock()
	defer tse.RUnlock()
	return tse.ready
}
