package health

import "sync/atomic"

type Health struct{ ready int32 }

func New() *Health { return &Health{} }

func (h *Health) SetReady(ready bool) {
	if ready { atomic.StoreInt32(&h.ready, 1) } else { atomic.StoreInt32(&h.ready, 0) }
}

func (h *Health) IsReady() bool { return atomic.LoadInt32(&h.ready) == 1 }
