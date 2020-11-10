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
	envMap map[string]Environment
	ready  bool
}

func NewEnvironments() *Environments {
	return &Environments{
		mu:     &sync.RWMutex{},
		envMap: make(map[string]Environment),
		ready:  false,
	}
}

func (e *Environments) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.envMap)
}

func (e *Environments) Names() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, e.Len())

	i := 0
	for name := range e.envMap {
		names[i] = name
		i++
	}

	return names
}

func (e *Environments) Values() []Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()

	vals := make([]Environment, e.Len())

	i := 0
	for _, val := range e.envMap {
		vals[i] = val
		i++
	}

	return vals
}

func (e *Environments) SetEnvironment(name string, env Environment) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.envMap[name] = env
}

func (e *Environments) RemoveEnvironment(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.envMap, name)
}

func (e *Environments) Environment(name string) Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.envMap[name]
}

func (e *Environments) SetReady(val bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ready = val
}

func (e *Environments) AreReady() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ready
}
