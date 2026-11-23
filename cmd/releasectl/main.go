package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/consize-oss/consize-prometheus-plugin/internal/release"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: releasectl keygen|package|verify")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	privatePath := fs.String("private-key", "", "base64 Ed25519 private key file")
	privateEnv := fs.String("private-key-env", "", "name of environment variable containing private key")
	publicPath := fs.String("public-key", "keys/publisher.pub", "trusted base64 Ed25519 public key")
	dir := fs.String("dir", "dist", "release directory")
	version := fs.String("version", "", "release version without v prefix")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	switch args[0] {
	case "keygen":
		if *privatePath == "" {
			return fmt.Errorf("private-key path required")
		}
		public, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		// Never replace a publisher key by accident, and never print secret material.
		for _, path := range []string{*privatePath, *publicPath} {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				return fmt.Errorf("key destination exists or is inaccessible: %s", path)
			}
		}
		if err := exclusiveWrite(*privatePath, []byte(base64.StdEncoding.EncodeToString(private)+"\n"), 0600); err != nil {
			return err
		}
		return exclusiveWrite(*publicPath, []byte(base64.StdEncoding.EncodeToString(public)+"\n"), 0644)
	case "package":
		var raw string
		if (*privatePath == "") == (*privateEnv == "") {
			return fmt.Errorf("choose one private-key or private-key-env")
		}
		if *privateEnv != "" {
			raw = os.Getenv(*privateEnv)
		} else {
			data, err := os.ReadFile(*privatePath)
			if err != nil {
				return err
			}
			raw = string(data)
		}
		key, err := release.ReadPrivate(raw)
		if err != nil {
			return err
		}
		trusted, err := marketplace.PublicKey(*publicPath)
		if err != nil {
			return err
		}
		if !key.Public().(ed25519.PublicKey).Equal(trusted) {
			return fmt.Errorf("signing key does not match trusted publisher key")
		}
		return release.Build(context.Background(), *dir, *version, key)
	case "verify":
		public, err := marketplace.PublicKey(*publicPath)
		if err != nil {
			return err
		}
		return release.Verify(*dir, public)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func exclusiveWrite(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
