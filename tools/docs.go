//go:build tools

package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/idebeijer/kubert/cmd"
)

func main() {
	kubertCmd := cmd.NewRootCmd()

	// delete old docs
	if err := os.RemoveAll("./docs"); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("./docs", 0755); err != nil {
		log.Fatal(err)
	}

	kubertCmd.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(kubertCmd.Command, "./docs"); err != nil {
		log.Fatal(err)
	}
}
