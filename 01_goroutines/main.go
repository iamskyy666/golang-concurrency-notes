package main

import (
	"fmt"
	"time"
)

func main() {

	start := time.Now()

	go func ()  {
		time.Sleep(300 * time.Millisecond)
		fmt.Println("✅ Goroutine A, finished, simulated API at",time.Since(start))
	}()

	go func ()  {
		time.Sleep(150 * time.Millisecond)
		fmt.Println("☑️ Goroutine B, finished, simulated API at",time.Since(start))
	}()

	// main() -> no waiting

	fmt.Println("main(): started 2 goroutines at..",time.Since(start))

	// small work -> any logic
	fmt.Println("main: doing step 1..",time.Since(start))
	time.Sleep(100 * time.Millisecond)

	fmt.Println("main: doing step 2..",time.Since(start))
	time.Sleep(100 * time.Millisecond)

	fmt.Println("main: doing step 3..",time.Since(start))
	time.Sleep(100 * time.Millisecond)

	// temp. sleep time to test
	time.Sleep(500 * time.Millisecond)

	// Exit..
	fmt.Println("main: exiting at..",time.Since(start))


	
}