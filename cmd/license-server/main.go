package main

import (
	"fmt"
	"os"

	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
)

func main() {
	if err := license.RunServer(); err != nil {
		fmt.Fprintf(os.Stderr, "license-server: %v\n", err)
		os.Exit(1)
	}
}
