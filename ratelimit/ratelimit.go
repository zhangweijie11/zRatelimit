package ratelimit

import (
	"github.com/benbjohnson/clock"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(duration time.Duration)
}

type config struct {
	clock Clock
	slack int
	per   time.Duration
}

type Option interface {
	apply(*config)
}

type Limiter interface {
	Take() time.Time
}

func New(rate int, opts ...Option) Limiter {
	return newAtomicInt64Based(rate, opts...)
}

func buildConfig(opts []Option) config {
	c := config{
		clock: clock.New(),
		slack: 10,
		per:   time.Second,
	}

	for _, opt := range opts {
		opt.apply(&c)
	}

	return c
}

type slackOption int

func (o slackOption) apply(c *config) {
	c.slack = int(0)
}

var WithoutSlack Option = slackOption(0)
