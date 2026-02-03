package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml" // Você precisará importar isso ou usar encoding/json
)

// GetResourceYAML busca o recurso e o converte para YAML
func GetResourceYAML(ctx context.Context, client *kubernetes.Clientset, ns, kind, name string) (string, error) {
	// K8s client-go não tem um "Get Generic" fácil sem usar dynamic client.
	// Para simplificar, vamos implementar switch cases para os tipos mais comuns.
	// Se precisar de TODOS os tipos, teríamos que usar client.Dynamic().

	var obj interface{}
	var err error

	switch kind {
	case "Pod":
		obj, err = client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	case "Service":
		obj, err = client.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	case "Deployment":
		obj, err = client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	case "StatefulSet":
		obj, err = client.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
	case "DaemonSet":
		obj, err = client.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
	case "ReplicaSet":
		obj, err = client.AppsV1().ReplicaSets(ns).Get(ctx, name, metav1.GetOptions{})
	case "Namespace":
		obj, err = client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	case "Node":
		obj, err = client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	default:
		return "", fmt.Errorf("tipo de recurso não suportado para visualização YAML: %s", kind)
	}

	if err != nil {
		return "", err
	}

	// Remove campos sujos do client-go (managedFields costumam poluir muito a view)
	// Isso é opcional, mas melhora a leitura.
	// Nota: Em Go puro sem reflection profunda é difícil limpar campos específicos de structs tipadas.
	// O mais simples é converter para JSON/YAML direto.

	// Converte objeto Go -> YAML
	// Usamos sigs.k8s.io/yaml que é o padrão do K8s, mas pode usar gopkg.in/yaml.v2
	y, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("erro ao converter para yaml: %w", err)
	}

	return string(y), nil
}

// GetPodLogs busca os logs de um pod (e container opcional)
func GetPodLogs(ctx context.Context, client *kubernetes.Clientset, ns, name, container string, tailLines int64) ([]string, error) {
	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := client.CoreV1().Pods(ns).GetLogs(name, opts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir stream de logs: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler logs: %w", err)
	}

	// Quebra em linhas
	lines := []string{}
	// Processamento simples de string (pode otimizar para logs gigantes)
	raw := buf.String()
	// Split manual ou usando strings.Split
	// Vamos fazer um split simples
	currentLine := ""
	for _, char := range raw {
		if char == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
		} else {
			currentLine += string(char)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines, nil
}