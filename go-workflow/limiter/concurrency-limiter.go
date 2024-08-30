package limiter

type ConcurrencyLimiter struct {
	tickets chan struct{}
}

func NewConcurrencyLimiter(maxConcurrency int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		tickets: make(chan struct{}, maxConcurrency),
	}
}

func (cl *ConcurrencyLimiter) Acquire() {
	cl.tickets <- struct{}{} // acquire a ticket
}

func (cl *ConcurrencyLimiter) Release() {
	<-cl.tickets // release a ticket
}
