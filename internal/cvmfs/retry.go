package cvmfs

import "time"

const (
	// Number of seconds for the initial wait duration
	defaultInitWait = 5
	// Number of seconds for the maximum wait duration
	defaultMaxWait = 1800
)

// Waiter implements an exponential backoff retry scheme: each successive call to Wait()
// will block for double the time of the previous call, up to a specified maximum time
type Waiter struct {
	currentWait int
	initWait    int
	maxWait     int
}

// DefaultWaiter constructs a Waiter object with the default wait values
func DefaultWaiter() Waiter {
	return Waiter{
		currentWait: defaultInitWait,
		initWait:    defaultInitWait,
		maxWait:     defaultMaxWait,
	}
}

// NewWaiter constructs a Waiter object with the specified initial and maximum
// wait values
func NewWaiter(initWait, maxWait int) Waiter {
	return Waiter{initWait, initWait, maxWait}
}

// Wait blocks for an amount of time as per the exponential backoff scheme
func (w *Waiter) Wait() {
	time.Sleep(time.Duration(w.currentWait) * time.Second)
	w.currentWait *= 2
	if w.currentWait > w.maxWait {
		w.currentWait = w.maxWait
	}
}

// Reset the state of the Waiter object. Next call to Wait will block for the initial
// wait duration
func (w *Waiter) Reset() {
	w.currentWait = w.initWait
}
