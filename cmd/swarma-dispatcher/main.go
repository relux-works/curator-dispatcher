package main

import (
	"fmt"
	"os"

	"github.com/relux-works/swarma-dispatcher/internal/transport"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "--config" {
		fmt.Fprintln(os.Stderr, "usage: swarma-dispatcher --config /absolute/server.json")
		os.Exit(1)
	}
	if err := transport.Daemon(os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "dispatcher refused:", err)
		os.Exit(1)
	}
}
