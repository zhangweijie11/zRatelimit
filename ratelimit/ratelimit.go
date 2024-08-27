package ratelimit

import (
	"context"
	"golang.org/x/time/rate"
	"sync/atomic"
	"time"
)

type Strategy uint8

const (
	None Strategy = iota
	LeakyBucket
)

// 用于计数器的递减操作
var minusOne = ^uint32(0)

type Limiter struct {
	strategy           Strategy
	maxCount           atomic.Uint32
	interval           time.Duration
	count              atomic.Uint32
	ticker             *time.Ticker
	tokens             chan struct{}
	ctx                context.Context
	cancelFunc         context.CancelFunc
	leakyBucketLimiter *rate.Limiter
}

func New(ctx context.Context, max uint, duration time.Duration) *Limiter {
	internalCtx, cancel := context.WithCancel(context.TODO())

	maxCount := &atomic.Uint32{}
	maxCount.Store(uint32(max))
	limiter := &Limiter{
		strategy:   None,
		interval:   duration,
		ticker:     time.NewTicker(duration),
		tokens:     make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	limiter.maxCount.Store(uint32(max))
	limiter.count.Store(uint32(max))

	go limiter.run(internalCtx)

	return limiter
}

func (limiter *Limiter) run(ctx context.Context) {
	defer close(limiter.tokens)
	for {
		if limiter.count.Load() == 0 {
			<-limiter.ticker.C
			limiter.count.Store(limiter.maxCount.Load())
		}
		select {
		case <-ctx.Done():
			limiter.ticker.Stop()
			return
		case <-limiter.ctx.Done():
			limiter.ticker.Stop()
			return
		case limiter.tokens <- struct{}{}:
			limiter.count.Add(minusOne)
		case <-limiter.ticker.C:
			limiter.count.Store(limiter.maxCount.Load())
		}
	}
}

func (limiter *Limiter) Take() {
	switch limiter.strategy {
	case LeakyBucket:
		_ = limiter.leakyBucketLimiter.Wait(context.TODO())
	default:
		<-limiter.tokens
	}
}
