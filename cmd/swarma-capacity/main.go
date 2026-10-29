package main

import (
	"fmt"
	"os"

	"github.com/relux-works/swarma-dispatcher/internal/transport"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "--config" {
		fmt.Fprintln(os.Stderr, "usage: swarma-capacity --config /absolute/client.json")
		os.Exit(1)
	}
	response, err := transport.Capacity(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "capacity unavailable:", err)
		os.Exit(1)
	}
	if _, err := fmt.Fprintln(os.Stdout, response); err != nil {
		fmt.Fprintln(os.Stderr, "capacity output failed:", err)
		os.Exit(1)
	}
	// A completed transport exchange still means closed admission.
	os.Exit(2)
}
