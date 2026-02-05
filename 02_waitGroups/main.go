package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(3) // wait for 3 goroutines

	go func() {
		defer wg.Done()
		fmt.Println("Task 1..")
		time.Sleep(250 * time.Millisecond)
		fmt.Println("Task 1 done.. ✅")
	}()

		go func() {
		defer wg.Done()
		fmt.Println("Task 2..")
		time.Sleep(150 * time.Millisecond)
		fmt.Println("Task 2 done.. ☑️")
	}()

		go func() {
		defer wg.Done()
		fmt.Println("Task 3..")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("Task 3 done.. ✔️")
	}()

	fmt.Println("main: waiting for all tasks to finish.. ⌛")
	
	wg.Wait()

	fmt.Println("main: All tasks finished. Good job ✨")
}