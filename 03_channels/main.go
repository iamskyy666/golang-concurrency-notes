package main

import (
	"fmt"
	"time"
)

func main() {
	type User struct {
		ID   int
		Name string
	}

	ch := make(chan User)

	// worker goroutine
	go func() {
		// simulate slow work.. Maybe an API call.
		time.Sleep(time.Millisecond * 200)

		// Send: Blocks until main() receives.
		// unbuffered channel - send + receives / handshake
		ch<-User{ID:100, Name: "Skyy"} //! sending val. to ch.
	}()

	fmt.Println("main: waiting to receive user... ⌛")
	
	u := <-ch //! receiving val. from ch.
	fmt.Println("main: now got user..", u,u.ID,u.Name)
}