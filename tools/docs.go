//go:build tools

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/idebeijer/kubert/cmd"
)

const docsDir = "./docs/commands"

func main() {
	kubertCmd := cmd.NewRootCmd()

	// Wipe and recreate only the generated subtree, leaving hand-written docs untouched.
	if err := os.RemoveAll(docsDir); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(docsDir, 0750); err != nil {
		log.Fatal(err)
	}

	kubertCmd.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(kubertCmd.Command, docsDir); err != nil {
		log.Fatal(err)
	}

	// Add "sh" to all example code blocks
	if err := addShellToCodeBlocks(docsDir); err != nil {
		log.Fatal(err)
	}

	// Copy kubert.md to README.md with a note so GitHub renders it when
	// browsing to docs/commands/.
	data, err := os.ReadFile(filepath.Join(docsDir, "kubert.md"))
	if err != nil {
		log.Fatal(err)
	}

	note := []byte("<!-- This file is a generated copy of kubert.md for GitHub display purposes -->\n" +
		"> [!NOTE]\n" +
		"> This file is a generated copy of `kubert.md`.\n\n")
	data = append(note, data...)

	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), data, 0600); err != nil {
		log.Fatal(err)
	}
}

func addShellToCodeBlocks(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".md" {
			continue
		}
		path := filepath.Join(dir, file.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		inCodeBlock := false
		inExamples := false
		for i, line := range lines {
			if strings.HasPrefix(line, "### Examples") {
				inExamples = true
			} else if strings.HasPrefix(line, "#") {
				inExamples = false
			}

			if strings.HasPrefix(line, "```") {
				if !inCodeBlock {
					if inExamples && (line == "```" || line == "```\r") {
						lines[i] = strings.Replace(line, "```", "```sh", 1)
					}
				}
				inCodeBlock = !inCodeBlock
			}
		}

		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(path, []byte(newContent), 0600); err != nil {
			return err
		}
	}
	return nil
}
