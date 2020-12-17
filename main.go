package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	// run process in the background, it will pass response into chan
	responseChan := make(chan *Response, 1)
	go func() {
		responseChan <- process(processCtx)
	}()

	// wait for stop signal from system or for response from process
	var response *Response
	select {
	case sig := <-termChan:
		processCancel()
		// wait until process returns
		<-responseChan
		response = &Response{Msg: fmt.Sprintf("Received OS signal %s. Cancelled", sig.String()), Failed: true}
	case r := <-responseChan:
		response = r
		// process finished, no need to cancel it
	}

	// print response to stdout
	text, err := json.Marshal(response)
	if err != nil {
		text, _ = json.Marshal(Response{Msg: "Invalid response object"})
	}
	fmt.Println(string(text))

	// exit program
	if response.Failed {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
