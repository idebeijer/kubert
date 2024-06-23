package kubeconfig

import (
	"fmt"
	"path/filepath"
)

// ContextLoader struct to load contexts from kubeconfigs
type ContextLoader struct {
	Loader *Loader
}

// NewContextLoader returns a new ContextLoader
func NewContextLoader(loader *Loader) *ContextLoader {
	return &ContextLoader{Loader: loader}
}

// LoadContexts method to load all contexts from all kubeconfigs
func (c *ContextLoader) LoadContexts() (map[string]WithPath, error) {
	allKubeconfigs, err := c.Loader.LoadAll()
	if err != nil {
		return nil, err
	}

	contexts := make(map[string]WithPath)
	for _, kubeconfig := range allKubeconfigs {
		for contextName := range kubeconfig.Config.Contexts {
			uniqueKey := fmt.Sprintf("%s", contextName)
			if _, ok := contexts[uniqueKey]; ok { // TODO: Find a better solution for this, as this might not be a good nor clean way to handle duplicates
				fmt.Printf("WARNING: Duplicate context name found: '%s'. Adding filename as suffix to prevent "+
					"overwrite, this will cause if there is another duplicate in the same file.\n", contextName)
				fmt.Println("Press enter to continue")
				fmt.Scanln()
				uniqueKey = fmt.Sprintf("%s::%s", contextName, filepath.Base(kubeconfig.FilePath))
			}
			contexts[uniqueKey] = kubeconfig
		}
	}
	return contexts, nil
}
