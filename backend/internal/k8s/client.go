package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient cria um clientset a partir de um kubeconfig em texto puro.
func buildConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar REST config: %w", err)
	}
	return cfg, nil
}

func buildConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		Precedence: []string{},
	}
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	// Usa clientcmd.BuildConfigFromKubeconfigReader com os bytes
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar REST config: %w", err)
	}
	return cfg, nil
}

