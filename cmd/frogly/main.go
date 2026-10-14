// Command frogly is a task planner for self-organized teams.
package main

import (
	"fmt"

	"github.com/the127/frogly/internal/version"
)

func main() {
	fmt.Println("frogly", version.Get())
}
