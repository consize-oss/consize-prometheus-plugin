package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/consize-oss/consize-prometheus-plugin/prometheus"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-manifest", "--manifest":
			if err := json.NewEncoder(os.Stdout).Encode(prometheus.NewWithClient(nil, prometheus.Config{}).Manifest()); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		case "--version":
			fmt.Println(prometheus.Version)
			return
		case "-h", "--help":
			fmt.Println("consize-prometheus-plugin [--manifest|--version]\nWithout flags: serve one Consize protocol 1 JSON request on stdin.")
			return
		}
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "unknown argument; use --help")
		os.Exit(1)
	}
	if err := marketplace.ServeOnce(context.Background(), os.Stdin, os.Stdout, prometheus.FromJSON); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
