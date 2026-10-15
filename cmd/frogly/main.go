// Command frogly is a task planner for self-organized teams.
package main

import (
	"fmt"
	"os"

	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/version"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "frogly:", err)
		os.Exit(1)
	}
}

func run() error {
	// nothing reads the configuration yet, but a broken one should stop
	// frogly before it does anything
	_, err := config.Load(os.Environ())
	if err != nil {
		return err
	}

	fmt.Println("frogly", version.Get())

	return nil
}
