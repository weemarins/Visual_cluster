package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient cria um clientset a partir de um kubeconfig em bytes.
func NewClient(kubeconfig []byte) (*kubernetes.Clientset, error) {
	config, err := buildConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func buildConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar REST config: %w", err)
	}

	return cfg, nil
}
