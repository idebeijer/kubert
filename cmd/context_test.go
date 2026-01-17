package cmd

import (
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestCreateTempKubeconfigFile_Isolation(t *testing.T) {
	// Create a temp file to act as the "original" complex kubeconfig
	tempFile, err := os.CreateTemp("", "original-kubeconfig-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatal(err)
		}
	}()

	// Define a complex configuration with multiple contexts, clusters, and users
	config := api.NewConfig()

	// Cluster 1 data
	config.Clusters["shared-cluster"] = &api.Cluster{Server: "https://shared.example.com"}

	config.AuthInfos["user-1"] = &api.AuthInfo{Token: "token-1"}
	config.Contexts["ctx-1"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-1", Namespace: "ns-1"}

	config.AuthInfos["user-2"] = &api.AuthInfo{Username: "admin", Password: "password"}
	config.Contexts["ctx-2"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-2", Namespace: "default"}

	// Cluster 2 data
	config.Clusters["other-cluster"] = &api.Cluster{Server: "https://other.example.com"}
	config.AuthInfos["user-3"] = &api.AuthInfo{ClientCertificate: "/path/to/cert"}
	config.Contexts["ctx-3"] = &api.Context{Cluster: "other-cluster", AuthInfo: "user-3"}

	// Write this original config to disk
	if err := clientcmd.WriteToFile(*config, tempFile.Name()); err != nil {
		t.Fatal(err)
	}

	// Now try to isolate "ctx-2"
	isolatedFile, cleanup, err := createTempKubeconfigFile(tempFile.Name(), "ctx-2", "")
	if err != nil {
		t.Fatalf("createTempKubeconfigFile failed: %v", err)
	}
	defer cleanup()

	// Load the generated isolated config to verify contents
	isolatedConfig, err := clientcmd.LoadFromFile(isolatedFile.Name())
	if err != nil {
		t.Fatalf("Failed to load generated isolated config: %v", err)
	}

	// Verify Context correctness
	if len(isolatedConfig.Contexts) != 1 {
		t.Errorf("Expected exactly 1 context, got %d", len(isolatedConfig.Contexts))
	}
	ctx, exists := isolatedConfig.Contexts["ctx-2"]
	if !exists {
		t.Fatal("Target context 'ctx-2' missing from isolated config")
	}
	if ctx.Cluster != "shared-cluster" {
		t.Errorf("Context cluster mismatch. Got %s, want shared-cluster", ctx.Cluster)
	}
	if ctx.AuthInfo != "user-2" {
		t.Errorf("Context user mismatch. Got %s, want user-2", ctx.AuthInfo)
	}

	// Verify Cluster correctness
	if len(isolatedConfig.Clusters) != 1 {
		t.Errorf("Expected exactly 1 cluster, got %d", len(isolatedConfig.Clusters))
	}
	cluster, exists := isolatedConfig.Clusters["shared-cluster"]
	if !exists {
		t.Fatal("Target cluster 'shared-cluster' missing from isolated config")
	}
	if cluster.Server != "https://shared.example.com" {
		t.Errorf("Cluster server mismatch. Got %s", cluster.Server)
	}

	// Verify User correctness
	if len(isolatedConfig.AuthInfos) != 1 {
		t.Errorf("Expected exactly 1 user, got %d", len(isolatedConfig.AuthInfos))
	}
	user, exists := isolatedConfig.AuthInfos["user-2"]
	if !exists {
		t.Fatal("Target user 'user-2' missing from isolated config")
	}
	if user.Username != "admin" {
		t.Errorf("User username mismatch. Got %s", user.Username)
	}

	// Verify leakage (ensure other data is NOT present)
	if _, exists := isolatedConfig.Clusters["other-cluster"]; exists {
		t.Error("Leaked 'other-cluster' into isolated config")
	}
	if _, exists := isolatedConfig.AuthInfos["user-3"]; exists {
		t.Error("Leaked 'user-3' into isolated config")
	}
}
