//go:build tools

package main

import (
	"log"

	"github.com/idebeijer/kubert/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	kubertCmd := cmd.NewRootCmd()

	if err := doc.GenMarkdownTree(kubertCmd.Command, "./docs"); err != nil {
		log.Fatal(err)
	}
}
