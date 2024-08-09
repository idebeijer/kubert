package kubeconfig

// ContextLoader struct to load contexts from kubeconfigs
type ContextLoader struct {
	Loader *Loader
}

// NewContextLoader returns a new ContextLoader
func NewContextLoader(loader *Loader) *ContextLoader {
	return &ContextLoader{Loader: loader}
}

type Context struct {
	Name string
	WithPath
}

// LoadContexts method to load all contexts from all kubeconfigs
func (c *ContextLoader) LoadContexts() ([]Context, error) {
	allKubeconfigs, err := c.Loader.LoadAll()
	if err != nil {
		return nil, err
	}

	var contexts []Context
	for _, kubeconfig := range allKubeconfigs {
		for contextName := range kubeconfig.Config.Contexts {
			contexts = append(contexts, Context{
				Name:     contextName,
				WithPath: kubeconfig,
			})
		}
	}

	return contexts, nil
}
