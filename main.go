package main

import (
	"fmt"
	"os"
)

func main() {
	proc, port, err := getscprocess.getProcessInfo(false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Process ID: %d, Port: %d\n", proc, port)
}
