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
	if err := os.MkdirAll("./docs", 0750); err != nil {
		log.Fatal(err)
	}

	kubertCmd.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(kubertCmd.Command, "./docs"); err != nil {
		log.Fatal(err)
	}

	// Copy kubert.md to README.md with a note
	data, err := os.ReadFile("./docs/kubert.md")
	if err != nil {
		log.Fatal(err)
	}

	note := []byte("<!-- This file is a generated copy of kubert.md for GitHub display purposes -->\n" +
		"> [!NOTE]\n" +
		"> This file is a generated copy of `kubert.md`.\n\n")
	data = append(note, data...)

	if err := os.WriteFile("./docs/README.md", data, 0600); err != nil {
		log.Fatal(err)
	}
}
