package util

import (
	"os"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func KubeClientConfig() (*api.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	clientConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return clientConfig, nil
}
