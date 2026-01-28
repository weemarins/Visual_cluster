package k8s

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GraphNode representa um recurso no grafo.
type GraphNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// GraphEdge representa uma relação entre recursos.
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// ClusterGraph representa o grafo completo retornado para o frontend.
type ClusterGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// BuildTopologyGraph descobre recursos e constrói o grafo.
func BuildTopologyGraph(ctx context.Context, client *kubernetes.Clientset, namespaceFilter string) (*ClusterGraph, error) {
	g := &ClusterGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	nsSelector := metav1.ListOptions{}
	if namespaceFilter != "" && namespaceFilter != "all" {
		// Vamos apenas usar o namespaceFilter quando listarmos recursos namespaced.
	}

	// Nodes (não namespaced)
	nodes, err := client.CoreV1().Nodes().List(timeoutCtx, metav1.ListOptions{})
	if err == nil {
		for _, n := range nodes.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:     "node:" + n.Name,
				Kind:   "Node",
				Name:   n.Name,
				Labels: n.Labels,
			})
		}
	}

	// Namespaces
	namespaces, err := client.CoreV1().Namespaces().List(timeoutCtx, nsSelector)
	if err == nil {
		for _, ns := range namespaces.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:   "ns:" + ns.Name,
				Kind: "Namespace",
				Name: ns.Name,
			})
		}
	}

	nsList := []string{}
	for _, ns := range namespaces.Items {
		if namespaceFilter == "" || namespaceFilter == "all" || namespaceFilter == ns.Name {
			nsList = append(nsList, ns.Name)
		}
	}

	// Para cada namespace relevante, buscamos os recursos e montamos edges básicos.
	for _, ns := range nsList {
		// Deployments
		deps, _ := client.AppsV1().Deployments(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, d := range deps.Items {
			depID := "deploy:" + ns + ":" + d.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        depID,
				Kind:      "Deployment",
				Name:      d.Name,
				Namespace: ns,
				Labels:    d.Labels,
			})
		}

		// StatefulSets
		stsList, _ := client.AppsV1().StatefulSets(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, s := range stsList.Items {
			stsID := "sts:" + ns + ":" + s.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        stsID,
				Kind:      "StatefulSet",
				Name:      s.Name,
				Namespace: ns,
				Labels:    s.Labels,
			})
		}

		// DaemonSets
		dsList, _ := client.AppsV1().DaemonSets(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, d := range dsList.Items {
			dID := "ds:" + ns + ":" + d.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        dID,
				Kind:      "DaemonSet",
				Name:      d.Name,
				Namespace: ns,
				Labels:    d.Labels,
			})
		}

		// ReplicaSets
		rsList, _ := client.AppsV1().ReplicaSets(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, rs := range rsList.Items {
			rsID := "rs:" + ns + ":" + rs.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        rsID,
				Kind:      "ReplicaSet",
				Name:      rs.Name,
				Namespace: ns,
				Labels:    rs.Labels,
			})
		}

		// Pods
		pods, _ := client.CoreV1().Pods(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, p := range pods.Items {
			podID := "pod:" + ns + ":" + p.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        podID,
				Kind:      "Pod",
				Name:      p.Name,
				Namespace: ns,
				Labels:    p.Labels,
			})
		}

		// Services
		svcs, _ := client.CoreV1().Services(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, s := range svcs.Items {
			svcID := "svc:" + ns + ":" + s.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        svcID,
				Kind:      "Service",
				Name:      s.Name,
				Namespace: ns,
				Labels:    s.Labels,
			})
		}

		// HPAs
		hpas, _ := client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(timeoutCtx, metav1.ListOptions{})
		for _, h := range hpas.Items {
			hpaID := "hpa:" + ns + ":" + h.Name
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        hpaID,
				Kind:      "HPA",
				Name:      h.Name,
				Namespace: ns,
				Labels:    h.Labels,
			})
		}

		// Edges simples: Service -> Pod (por selector), Deployment -> ReplicaSet -> Pod, StatefulSet -> Pod, HPA -> Deployment/StatefulSet.
		buildEdgesForNamespace(g, ns, deps, stsList, dsList, rsList, pods, svcs, hpas)
	}

	return g, nil
}

func buildEdgesForNamespace(
	g *ClusterGraph,
	ns string,
	deps *appsv1.DeploymentList,
	stsList *appsv1.StatefulSetList,
	dsList *appsv1.DaemonSetList,
	rsList *appsv1.ReplicaSetList,
	pods *corev1.PodList,
	svcs *corev1.ServiceList,
	hpas *autoscalingv2.HorizontalPodAutoscalerList,
) {
	// Service -> Pod
	for _, svc := range svcs.Items {
		for _, pod := range pods.Items {
			if podMatchesSelector(pod.Labels, svc.Spec.Selector) {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:svc->pod:" + ns + ":" + svc.Name + "->" + pod.Name,
					Source: "svc:" + ns + ":" + svc.Name,
					Target: "pod:" + ns + ":" + pod.Name,
				})
			}
		}
	}

	// Deployment -> ReplicaSet -> Pod
	for _, dep := range deps.Items {
		for _, rs := range rsList.Items {
			if ownerRefMatches(rs.OwnerReferences, "Deployment", dep.Name) {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:deploy->rs:" + ns + ":" + dep.Name + "->" + rs.Name,
					Source: "deploy:" + ns + ":" + dep.Name,
					Target: "rs:" + ns + ":" + rs.Name,
				})
				for _, pod := range pods.Items {
					if ownerRefMatches(pod.OwnerReferences, "ReplicaSet", rs.Name) {
						g.Edges = append(g.Edges, GraphEdge{
							ID:     "edge:rs->pod:" + ns + ":" + rs.Name + "->" + pod.Name,
							Source: "rs:" + ns + ":" + rs.Name,
							Target: "pod:" + ns + ":" + pod.Name,
						})
					}
				}
			}
		}
	}

	// StatefulSet -> Pod
	for _, sts := range stsList.Items {
		for _, pod := range pods.Items {
			if ownerRefMatches(pod.OwnerReferences, "StatefulSet", sts.Name) {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:sts->pod:" + ns + ":" + sts.Name + "->" + pod.Name,
					Source: "sts:" + ns + ":" + sts.Name,
					Target: "pod:" + ns + ":" + pod.Name,
				})
			}
		}
	}

	// DaemonSet -> Pod
	for _, ds := range dsList.Items {
		for _, pod := range pods.Items {
			if ownerRefMatches(pod.OwnerReferences, "DaemonSet", ds.Name) {
				g.Edges = append(g.Edges, GraphEdge{
					ID:     "edge:ds->pod:" + ns + ":" + ds.Name + "->" + pod.Name,
					Source: "ds:" + ns + ":" + ds.Name,
					Target: "pod:" + ns + ":" + pod.Name,
				})
			}
		}
	}

	// HPA -> Deployment/StatefulSet
	for _, h := range hpas.Items {
		targetRef := h.Spec.ScaleTargetRef
		switch targetRef.Kind {
		case "Deployment":
			g.Edges = append(g.Edges, GraphEdge{
				ID:     "edge:hpa->deploy:" + ns + ":" + h.Name + "->" + targetRef.Name,
				Source: "hpa:" + ns + ":" + h.Name,
				Target: "deploy:" + ns + ":" + targetRef.Name,
			})
		case "StatefulSet":
			g.Edges = append(g.Edges, GraphEdge{
				ID:     "edge:hpa->sts:" + ns + ":" + h.Name + "->" + targetRef.Name,
				Source: "hpa:" + ns + ":" + h.Name,
				Target: "sts:" + ns + ":" + targetRef.Name,
			})
		}
	}
}

func podMatchesSelector(labels, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func ownerRefMatches(refs []metav1.OwnerReference, kind, name string) bool {
	for _, r := range refs {
		if r.Kind == kind && r.Name == name {
			return true
		}
	}
	return false
}

