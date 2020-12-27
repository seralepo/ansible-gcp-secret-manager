package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// termChan will be used to receive kill signals from system
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	// create context for cancelling process
	processCtx, processCancel := context.WithCancel(context.Background())
	defer processCancel()

	// run process in the background, it will pass exit code into chan
	exitCodeChan := make(chan int, 1)
	go func() {
		exitCodeChan <- process(processCtx)
	}()

	// wait for stop signal from system or for finish of process
	var exitCode int
	select {
	case <-termChan:
		// if signal received from OS, cancel process and wait for it to finish
		processCancel()
		<-exitCodeChan
		exitCode = 143
	case c := <-exitCodeChan:
		exitCode = c
		// process finished, no need to cancel it
	}

	// exit program
	os.Exit(exitCode)
}
