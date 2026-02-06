package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(),450*time.Millisecond)
	defer cancel()
	slowWork(ctx)

	// main waits until ctx ends
	<-ctx.Done()
	fmt.Println("main: context ended.. ",ctx.Err())

	fmt.Println("main:exit")
}

func slowWork(ctx context.Context){
	select{
	case <-time.After(700*time.Millisecond):
		return
	case <-ctx.Done():
		return	
	}
}