// goose CLI 入口。
package main

import (
	"fmt"
	"os"

	"github.com/aaif-goose/gogo/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return cli.Execute()
}
