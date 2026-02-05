package main

import (
	"fmt"
	"time"
)

func main() {

	// ch can store 2 jobs without a receiver
	jobs := make(chan string, 2) // buffered channel

	go func() {
		fmt.Println("Producer.. Sending job 1!")
		jobs<-"job 1"

		fmt.Println("Producer.. Sending job 2!")
		jobs<-"job 2"

		// Now, buffer is full..
		fmt.Println("Producer.. Sending job 3, but this will wait until consumer reads!")
		jobs<-"job 3"

		fmt.Println("producer: sent all jobs ✅")
		close(jobs) // close the sender -> no more jobs left
	}()

		// output
		for job := range jobs{
			fmt.Println("Consumer got - ",job)
			time.Sleep(300 * time.Millisecond)

			fmt.Println("Consuming finised! ☑️",job)
		}

		fmt.Println("main: All jobs completed! ✔️")
}