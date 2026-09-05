package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/appkernia/appkernia/server/internal/command"
)

func main() {
	if err := command.Run("ak-cli", os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var usageErr *command.UsageError
		if errors.As(err, &usageErr) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
