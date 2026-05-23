package health

import "sync/atomic"

// Health tracks the readiness state of the service.
type Health struct {
	ready int32
}

// New returns a Health instance marked as not yet ready.
func New() *Health {
	return &Health{}
}

// SetReady marks the service as ready to serve traffic.
func (h *Health) SetReady(ready bool) {
	if ready {
		atomic.StoreInt32(&h.ready, 1)
	} else {
		atomic.StoreInt32(&h.ready, 0)
	}
}

// IsReady returns true if the service is ready.
func (h *Health) IsReady() bool {
	return atomic.LoadInt32(&h.ready) == 1
}
