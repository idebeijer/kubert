package kubert

import (
	"fmt"
	"os"
)

const (
	// ShellActiveEnvVar is the environment variable that is set to indicate that Kubert is active.
	ShellActiveEnvVar = "KUBERT_SHELL_ACTIVE"

	// ShellKubeconfigEnvVar is the environment variable that is set to the path of the temporary kubeconfig file, which is a working copy of the original kubeconfig file.
	ShellKubeconfigEnvVar = "KUBERT_SHELL_KUBECONFIG"

	// ShellStateFilePathEnvVar is the environment variable that is set to the path of the state file.
	ShellStateFilePathEnvVar = "KUBERT_SHELL_STATE_FILE"

	// ShellOriginalKubeconfigEnvVar is the environment variable that is set to the path of the original kubeconfig file.
	ShellOriginalKubeconfigEnvVar = "KUBERT_SHELL_ORIGINAL_KUBECONFIG"
)

// ShellPreFlightCheck checks if the shell was started by Kubert and if the kubeconfig file is the same as the one set by Kubert.
func ShellPreFlightCheck() error {
	if kubertActive := os.Getenv(ShellActiveEnvVar); kubertActive != "1" {
		return fmt.Errorf("shell not started by kubert")
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file not found at %s", kubeconfigPath)
	}

	kubertKubeconfig := os.Getenv(ShellKubeconfigEnvVar)
	if kubertKubeconfig == "" {
		return fmt.Errorf("kubeconfig file not found in environment")
	}

	// Check if the kubeconfig file is the same as the one set by kubert,
	// if not, it means that the user or some other process has changed the KUBECONFIG environment variable
	// and kubert should not interfere with it, so kubert will choose to exit instead
	if kubertKubeconfig != kubeconfigPath {
		return fmt.Errorf("KUBECONFIG environment variable does not match kubert kubeconfig," +
			" to prevent kubert from interfering with your original kubeconfigs, please start a new shell with kubert")
	}

	return nil
}
