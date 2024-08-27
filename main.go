package main

import (
	"context"
	"fmt"
	"github.com/zhangweijie11/zRatelimit/ratelimit"
	"sync"
	"time"
)

var wg sync.WaitGroup

func Example_default() {
	rl := ratelimit.New(context.Background(), 5, time.Second) // per second, some slack

	for i := 0; i < 10; i++ {
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

func main() {
	Example_default()
	//Example_withoutSlack()
}
