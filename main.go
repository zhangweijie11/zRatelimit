package main

import (
	"fmt"
	"github.com/zhangweijie11/zRatelimit/ratelimit"
	"sync"
	"time"
)

var wg sync.WaitGroup

func Example_default() {
	rl := ratelimit.New(100) // per second, some slack.

	rl.Take() // Initialize.
	//time.Sleep(time.Millisecond * 45) // Let some time pass.

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			rl.Take()
			defer wg.Done()
			fmt.Println("------------>11111111", i, time.Now())
			time.Sleep(1 * time.Second)
		}(i)
	}

	wg.Wait()

	// Output:
	// 1 0s
	// 2 0s
	// 3 0s
	// 4 4ms
	// 5 10ms
	// 6 10ms
	// 7 10ms
	// 8 10ms
	// 9 10ms
}

func Example_withoutSlack() {
	rl := ratelimit.New(100, ratelimit.WithoutSlack) // per second, no slack.

	prev := time.Now()
	for i := 0; i < 6; i++ {
		now := rl.Take()
		if i > 0 {
			fmt.Println(i, now.Sub(prev))
		}
		prev = now
	}

	// Output:
	// 1 10ms
	// 2 10ms
	// 3 10ms
	// 4 10ms
	// 5 10ms
}

func main() {
	Example_default()
	//Example_withoutSlack()
}
