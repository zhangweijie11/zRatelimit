package ratelimit

import (
	"context"
	"golang.org/x/time/rate"
	"math"
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

// CanTake 检查限速器是否有 Token
func (limiter *Limiter) CanTake() bool {
	switch limiter.strategy {
	case LeakyBucket:
		return limiter.leakyBucketLimiter.Tokens() > 0
	default:
		return limiter.count.Load() > 0
	}
}

func (limiter *Limiter) GetLimit() uint {
	return uint(limiter.maxCount.Load())
}

func (limiter *Limiter) SetLimit(max uint) {
	limiter.maxCount.Store(uint32(max))
	switch limiter.strategy {
	case LeakyBucket:
		limiter.leakyBucketLimiter.SetBurst(int(max))
	default:

	}
}

func (limiter *Limiter) Stop() {
	switch limiter.strategy {
	case LeakyBucket:
	default:
		if limiter.cancelFunc != nil {
			limiter.cancelFunc()
		}
	}
}

func (limiter *Limiter) SetDuration(d time.Duration) {
	limiter.interval = d
	switch limiter.strategy {
	case LeakyBucket:
		limiter.leakyBucketLimiter.SetLimit(rate.Every(d))
	default:
		limiter.ticker.Reset(d)
	}
}

// NewUnlimited create a bucket with approximated unlimited tokens
func NewUnlimited(ctx context.Context) *Limiter {
	internalctx, cancel := context.WithCancel(context.TODO())
	limiter := &Limiter{
		ticker:     time.NewTicker(time.Millisecond),
		tokens:     make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}
	limiter.maxCount.Store(math.MaxUint32)
	limiter.count.Store(math.MaxUint32)
	go limiter.run(internalctx)

	return limiter
}

// NewUnlimited create a bucket with approximated unlimited tokens
func NewLeakyBucket(ctx context.Context, max uint, duration time.Duration) *Limiter {
	limiter := &Limiter{
		strategy:           LeakyBucket,
		leakyBucketLimiter: rate.NewLimiter(rate.Every(duration), int(max)),
	}
	limiter.maxCount.Store(uint32(max))
	limiter.interval = duration
	return limiter
}
