package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	var once sync.Once

	workers:=5

	for i:=1; i<=workers; i++{
		wg.Add(1)

		go func(id int) {
			defer wg.Done();
			once.Do(setup)
			fmt.Println("worker",id, ": running")
		}(i)
	}

	wg.Wait();
	fmt.Println("main: done")
}

func setup(){
	fmt.Println("init: starting..")
	time.Sleep(600 * time.Millisecond)
	fmt.Println("init: done!")
}