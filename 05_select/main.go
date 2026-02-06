package main

import (
	"fmt"
	"time"
)

func main() {
	// select -> switch for channels

	resultCh := make(chan string)

	// worker goroutine
	go func() {
		// simulate slow work/network call
		time.Sleep(time.Millisecond * 240)

		resultCh<-"worker: success ✅"
	}()

	// timeout channel
	timeoutCh:=time.After(250 * time.Millisecond)

	select{
	case resp:= <- resultCh:
		fmt.Println("main: go result",resp)
	case <-timeoutCh:
		fmt.Println("main:timeout happened, stop waiting..")	
	}

	fmt.Println("main: work is now done ☑️")
}