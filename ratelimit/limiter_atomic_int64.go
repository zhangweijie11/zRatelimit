package ratelimit

import (
	"sync/atomic"
	"time"
)

type atomicInt64Limiter struct {
	prepadding  [64]byte      // 缓存行
	state       int64         // 下一个能够获取到令牌的时间间隔
	postpadding [56]byte      // 缓存行大小
	perRequest  time.Duration // 单次请求时间间隔
	maxSlack    time.Duration // 最大允许的时间间隔
	clock       Clock         // 模拟时间
}

func newAtomicInt64Based(rate int, opts ...Option) *atomicInt64Limiter {
	config := buildConfig(opts)
	perRequest := config.per / time.Duration(rate)
	l := &atomicInt64Limiter{
		perRequest: perRequest,
		maxSlack:   time.Duration(config.slack) * perRequest,
		clock:      config.clock,
	}

	atomic.StoreInt64(&l.state, 0)

	return l
}

// Take 确保多个 Take 调用之间花费的时间是平均时间
func (t *atomicInt64Limiter) Take() time.Time {
	var (
		newTImeOfNextPermissionIssue int64
		now                          int64
	)
	for {
		now = t.clock.Now().Unix()
		timeOfNextPermissionIssue := atomic.LoadInt64(&t.state)

		switch {
		case timeOfNextPermissionIssue == 0 || (t.maxSlack == 0 && now-timeOfNextPermissionIssue > int64(t.perRequest)):
			// 如果这是我们第一次调用或 t.maxSlack == 0，我们需要将下次执行时间设置为当前时间
			newTImeOfNextPermissionIssue = now
		case t.maxSlack > 0 && now-timeOfNextPermissionIssue > int64(t.maxSlack)+int64(t.perRequest):
			// 当前时间相较于上次许可执行时间的差值大于最大允许时间间隔和单次时间间隔之和，将下次许可执行时间设置为当前时间减去最大允许时间间隔
			newTImeOfNextPermissionIssue = now - int64(t.maxSlack)
		default:
			// 下次许可执行时间设置为上次许可执行时间加上单次时间间隔
			newTImeOfNextPermissionIssue = timeOfNextPermissionIssue + int64(t.perRequest)
		}
		// 判断&t.state是否等于timeOfNextPermissionIssue，如果等于则更新为newTImeOfNextPermissionIssue，如果不等于则不更新
		if atomic.CompareAndSwapInt64(&t.state, timeOfNextPermissionIssue, newTImeOfNextPermissionIssue) {
			break
		}
	}
	sleepDuration := time.Duration(newTImeOfNextPermissionIssue - now)
	if sleepDuration > 0 {
		t.clock.Sleep(sleepDuration)
		return time.Unix(0, newTImeOfNextPermissionIssue)
	}

	// 如果我们不想 atomicLimiter 休眠，则现在返回
	return time.Unix(0, now)
}
