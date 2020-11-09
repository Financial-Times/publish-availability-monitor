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
	mu     *sync.RWMutex
	EnvMap map[string]Environment
	ready  bool
}

func NewEnvironments() *Environments {
	return &Environments{
		mu:     &sync.RWMutex{},
		EnvMap: make(map[string]Environment),
		ready:  false,
	}
}

func (e *Environments) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.EnvMap)
}

func (e *Environments) Names() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, e.Len())

	i := 0
	for name := range e.EnvMap {
		names[i] = name
		i++
	}

	return names
}

func (e *Environments) Environment(name string) Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.EnvMap[name]
}

func (e *Environments) AreReady() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ready
}
