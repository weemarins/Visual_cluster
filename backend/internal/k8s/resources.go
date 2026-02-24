package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

// GetResourceYAML busca qualquer recurso Kubernetes (incluindo CRDs)
// usando o dynamic client e o RESTMapper para descobrir o GVR.
func GetResourceYAML(ctx context.Context, restCfg *rest.Config, ns, kind, name string) (string, error) {
	// Cria discovery client e mapper
	disco, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return "", fmt.Errorf("erro ao criar discovery client: %w", err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))

	// Tentativas de normalizar o 'kind' recebido: pode ser "pod", "pods", "Pod"
	candidates := []string{kind, strings.Title(kind)}
	if strings.HasSuffix(strings.ToLower(kind), "s") {
		// tenta singularizar simples
		candidates = append(candidates, strings.TrimSuffix(strings.Title(kind), "s"))
	} else {
		candidates = append(candidates, kind+"s")
	}

	var mapping *meta.RESTMapping
	for _, cand := range candidates {
		gk := schema.GroupKind{Kind: cand}
		m, err := mapper.RESTMapping(gk, "")
		if err == nil {
			mapping = m
			break
		}
	}

	if mapping == nil {
		return "", fmt.Errorf("tipo de recurso não suportado ou não encontrado: %s", kind)
	}

	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return "", fmt.Errorf("erro ao criar dynamic client: %w", err)
	}

	var res *unstructured.Unstructured
	resource := dyn.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if ns == "all" || ns == "" {
			return "", fmt.Errorf("namespace obrigatório para recursos com escopo de namespace")
		}
		resObj, err := resource.Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		res = resObj
	} else {
		resObj, err := resource.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		res = resObj
	}

	// Converter para YAML
	jsonBytes, err := res.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("erro ao converter recurso para json: %w", err)
	}
	y, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return "", fmt.Errorf("erro ao converter json->yaml: %w", err)
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
	raw := buf.String()
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


