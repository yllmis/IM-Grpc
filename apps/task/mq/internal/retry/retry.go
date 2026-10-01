package retry

import (
	"context"
	"time"
)

type Policy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func (p Policy) Run(ctx context.Context, fn func(attempt int) error, onRetry func()) error {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.InitialDelay <= 0 {
		p.InitialDelay = 100 * time.Millisecond
	}
	if p.MaxDelay < p.InitialDelay {
		p.MaxDelay = p.InitialDelay
	}
	var err error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		err = fn(attempt)
		if err == nil {
			return nil
		}
		if attempt == p.MaxAttempts {
			break
		}
		if onRetry != nil {
			onRetry()
		}
		delay := p.InitialDelay
		for i := 1; i < attempt && delay < p.MaxDelay; i++ {
			delay *= 2
			if delay > p.MaxDelay {
				delay = p.MaxDelay
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
