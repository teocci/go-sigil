// Package sample is a fixture used by the Go parser tests.
package sample

import "fmt"

// MaxRetries is the maximum number of retry attempts.
const MaxRetries = 3

// DefaultTimeout holds the default timeout in seconds.
var DefaultTimeout = 30

// Processor defines the processing contract.
type Processor interface {
	Process(data []byte) error
}

// Worker performs background work.
type Worker struct {
	name string
}

// NewWorker creates a Worker with the given name.
func NewWorker(name string) *Worker {
	return &Worker{name: name}
}

// Run starts the worker processing loop.
func (w *Worker) Run() {
	fmt.Println(w.name)
}

// helper is an unexported helper function.
func helper(x, y int) int {
	return x + y
}
