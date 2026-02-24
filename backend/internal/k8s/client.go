package k8s

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient cria um clientset e retorna também a *rest.Config
// a partir de um kubeconfig em bytes. O *rest.Config é útil
// para criar clientes dinâmicos (dynamic.Interface) quando
// precisamos operar com recursos genéricos.
func NewClient(kubeconfig []byte) (*kubernetes.Clientset, *rest.Config, error) {
	cfg, err := buildConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	// validação rápida criando um dynamic client (não usado aqui, apenas para garantir)
	_, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao criar dynamic client: %w", err)
	}

	return cs, cfg, nil
}

func buildConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar REST config: %w", err)
	}

	return cfg, nil
}
