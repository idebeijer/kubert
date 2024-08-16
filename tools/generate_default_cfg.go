package main

import (
	"fmt"

	"github.com/idebeijer/kubert/internal/config"
)

func main() {
	yamlConfig, err := config.GenerateDefaultYAML()
	if err != nil {
		panic(err)
	}
	fmt.Println(yamlConfig)
}
