package k8s

import (
	"context"
	"log"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

/*
========================
 MODELO DE DOMÍNIO (K8s)
========================
*/

type GraphNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type ClusterGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

/*
========================
 MODELO DE VISUALIZAÇÃO (React Flow)
========================
*/

type RFNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type,omitempty"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

type RFEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type RFGraph struct {
	Nodes []RFNode `json:"nodes"`
	Edges []RFEdge `json:"edges"`
}

/*
========================
 BUILD TOPOLOGY GRAPH
========================
*/

func BuildTopologyGraph(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespaceFilter string,
) (*RFGraph, error) {

	g := &ClusterGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	// Timeout de segurança
	timeoutCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex // Protege a escrita no grafo

	// Estruturas para armazenar os dados brutos (usado para criar as arestas depois)
	var (
		allDeps *appsv1.DeploymentList
		allSts  *appsv1.StatefulSetList
		allDs   *appsv1.DaemonSetList
		allRs   *appsv1.ReplicaSetList
		allPods *corev1.PodList
		allSvcs *corev1.ServiceList
		allHpas *autoscalingv2.HorizontalPodAutoscalerList
	)

	// Função auxiliar para tratamento de erro e filtro
	listOpts := metav1.ListOptions{}
	// Se tiver filtro de namespace específico, usamos ele. Se for "all" ou vazio, pegamos tudo ("").
	targetNS := ""
	if namespaceFilter != "all" && namespaceFilter != "" {
		targetNS = namespaceFilter
	}

	// ---------------------------------------------------------
	// 1. BUSCA PARALELA DE TODOS OS RECURSOS (CLUSTER-WIDE)
	// ---------------------------------------------------------
	// Isso faz apenas ~7 chamadas no total ao invés de N_namespaces * 7
	
	wg.Add(7)

	go func() {
		defer wg.Done()
		list, err := client.AppsV1().Deployments(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allDeps = list
			for _, d := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "deploy:" + d.Namespace + ":" + d.Name, Kind: "Deployment", Name: d.Name, Namespace: d.Namespace, Labels: d.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.AppsV1().StatefulSets(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allSts = list
			for _, s := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "sts:" + s.Namespace + ":" + s.Name, Kind: "StatefulSet", Name: s.Name, Namespace: s.Namespace, Labels: s.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.AppsV1().DaemonSets(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allDs = list
			for _, d := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "ds:" + d.Namespace + ":" + d.Name, Kind: "DaemonSet", Name: d.Name, Namespace: d.Namespace, Labels: d.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.AppsV1().ReplicaSets(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allRs = list
			for _, rs := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "rs:" + rs.Namespace + ":" + rs.Name, Kind: "ReplicaSet", Name: rs.Name, Namespace: rs.Namespace, Labels: rs.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.CoreV1().Pods(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allPods = list
			for _, p := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "pod:" + p.Namespace + ":" + p.Name, Kind: "Pod", Name: p.Name, Namespace: p.Namespace, Labels: p.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.CoreV1().Services(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allSvcs = list
			for _, s := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "svc:" + s.Namespace + ":" + s.Name, Kind: "Service", Name: s.Name, Namespace: s.Namespace, Labels: s.Labels})
			}
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		list, err := client.AutoscalingV2().HorizontalPodAutoscalers(targetNS).List(timeoutCtx, listOpts)
		if err == nil {
			mu.Lock()
			allHpas = list
			for _, h := range list.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "hpa:" + h.Namespace + ":" + h.Name, Kind: "HPA", Name: h.Name, Namespace: h.Namespace, Labels: h.Labels})
			}
			mu.Unlock()
		}
	}()

	// Nodes físicos e Namespaces (rápido)
	go func() {
		nodes, _ := client.CoreV1().Nodes().List(timeoutCtx, metav1.ListOptions{})
		if nodes != nil {
			mu.Lock()
			for _, n := range nodes.Items {
				g.Nodes = append(g.Nodes, GraphNode{ID: "node:" + n.Name, Kind: "Node", Name: n.Name, Labels: n.Labels})
			}
			mu.Unlock()
		}
	}()

	wg.Wait()

	// ---------------------------------------------------------
	// 2. CONSTRUÇÃO DE ARESTAS (EDGES)
	// ---------------------------------------------------------
	// Agora que temos tudo em memória, processamos as conexões.
	// Precisamos iterar por namespace para não conectar recursos de namespaces diferentes.

	// Agrupa recursos por namespace em mapas para acesso rápido
	// Isso evita loops aninhados gigantescos O(N^2) global
	podsByNs := make(map[string][]corev1.Pod)
	if allPods != nil {
		for _, p := range allPods.Items {
			podsByNs[p.Namespace] = append(podsByNs[p.Namespace], p)
		}
	}

	rsByNs := make(map[string][]appsv1.ReplicaSet)
	if allRs != nil {
		for _, rs := range allRs.Items {
			rsByNs[rs.Namespace] = append(rsByNs[rs.Namespace], rs)
		}
	}

	// Processa Edges
	if allSvcs != nil {
		for _, svc := range allSvcs.Items {
			nsPods := podsByNs[svc.Namespace]
			for _, pod := range nsPods {
				if podMatchesSelector(pod.Labels, svc.Spec.Selector) {
					g.Edges = append(g.Edges, GraphEdge{
						ID:     "edge:svc->pod:" + svc.Namespace + ":" + svc.Name + "->" + pod.Name,
						Source: "svc:" + svc.Namespace + ":" + svc.Name,
						Target: "pod:" + svc.Namespace + ":" + pod.Name,
					})
				}
			}
		}
	}

	if allDeps != nil {
		for _, dep := range allDeps.Items {
			nsRs := rsByNs[dep.Namespace]
			nsPods := podsByNs[dep.Namespace]
			
			for _, rs := range nsRs {
				if ownerRefMatches(rs.OwnerReferences, "Deployment", dep.Name) {
					g.Edges = append(g.Edges, GraphEdge{
						ID:     "edge:deploy->rs:" + dep.Namespace + ":" + dep.Name + "->" + rs.Name,
						Source: "deploy:" + dep.Namespace + ":" + dep.Name,
						Target: "rs:" + dep.Namespace + ":" + rs.Name,
					})
					// RS -> Pod
					for _, pod := range nsPods {
						if ownerRefMatches(pod.OwnerReferences, "ReplicaSet", rs.Name) {
							g.Edges = append(g.Edges, GraphEdge{
								ID:     "edge:rs->pod:" + dep.Namespace + ":" + rs.Name + "->" + pod.Name,
								Source: "rs:" + dep.Namespace + ":" + rs.Name,
								Target: "pod:" + dep.Namespace + ":" + pod.Name,
							})
						}
					}
				}
			}
		}
	}

	if allSts != nil {
		for _, sts := range allSts.Items {
			nsPods := podsByNs[sts.Namespace]
			for _, pod := range nsPods {
				if ownerRefMatches(pod.OwnerReferences, "StatefulSet", sts.Name) {
					g.Edges = append(g.Edges, GraphEdge{
						ID:     "edge:sts->pod:" + sts.Namespace + ":" + sts.Name + "->" + pod.Name,
						Source: "sts:" + sts.Namespace + ":" + sts.Name,
						Target: "pod:" + sts.Namespace + ":" + pod.Name,
					})
				}
			}
		}
	}

	if allDs != nil {
		for _, ds := range allDs.Items {
			nsPods := podsByNs[ds.Namespace]
			for _, pod := range nsPods {
				if ownerRefMatches(pod.OwnerReferences, "DaemonSet", ds.Name) {
					g.Edges = append(g.Edges, GraphEdge{
						ID:     "edge:ds->pod:" + ds.Namespace + ":" + ds.Name + "->" + pod.Name,
						Source: "ds:" + ds.Namespace + ":" + ds.Name,
						Target: "pod:" + ds.Namespace + ":" + pod.Name,
					})
				}
			}
		}
	}

	if allHpas != nil {
		for _, h := range allHpas.Items {
			ref := h.Spec.ScaleTargetRef
			ns := h.Namespace
			if ref.Kind == "Deployment" {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:hpa->deploy:" + ns + ":" + h.Name + "->" + ref.Name,
					Source: "hpa:" + ns + ":" + h.Name,
					Target: "deploy:" + ns + ":" + ref.Name,
				})
			} else if ref.Kind == "StatefulSet" {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:hpa->sts:" + ns + ":" + h.Name + "->" + ref.Name,
					Source: "hpa:" + ns + ":" + h.Name,
					Target: "sts:" + ns + ":" + ref.Name,
				})
			}
		}
	}

	log.Printf("[TOPOLOGY] Completed. Nodes: %d, Edges: %d", len(g.Nodes), len(g.Edges))

	// Conversão para React Flow
	rfNodes := []RFNode{}
	x, y := 0.0, 0.0
	for _, n := range g.Nodes {
		rfNodes = append(rfNodes, RFNode{
			ID:       n.ID,
			Type:     "default",
			Position: map[string]float64{"x": x, "y": y},
			Data: map[string]interface{}{
				"label":     n.Kind + ": " + n.Name,
				"namespace": n.Namespace, // Essencial
				"kind":      n.Kind,
				"labels":    n.Labels,
			},
		})
		y += 10
	}

	rfEdges := []RFEdge{}
	for _, e := range g.Edges {
		rfEdges = append(rfEdges, RFEdge{ID: e.ID, Source: e.Source, Target: e.Target})
	}

	return &RFGraph{Nodes: rfNodes, Edges: rfEdges}, nil
}

// Helpers
func podMatchesSelector(labels, selector map[string]string) bool {
	if len(selector) == 0 { return false }
	for k, v := range selector {
		if labels[k] != v { return false }
	}
	return true
}

func ownerRefMatches(refs []metav1.OwnerReference, kind, name string) bool {
	for _, r := range refs {
		if r.Kind == kind && r.Name == name { return true }
	}
	return false
}