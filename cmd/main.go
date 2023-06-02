package main

import (
	"fmt"
	"os"

	"github.com/innerspirit/getscprocess/lib"
)

func main() {
	proc, port, err := lib.GetProcessInfo(false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Process ID: %d, Port: %d\n", proc, port)
}
