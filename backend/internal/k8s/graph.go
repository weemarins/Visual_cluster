package k8s

import (
	"context"
	"time"
	"log"

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

	// -------- Grafo de domínio --------
	g := &ClusterGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// -------- Nodes (cluster-wide) --------
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

	// -------- Namespaces --------
	namespaces, err := client.CoreV1().Namespaces().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return &RFGraph{Nodes: []RFNode{}, Edges: []RFEdge{}}, nil
	}

	nsList := []string{}
	for _, ns := range namespaces.Items {
		if namespaceFilter == "" || namespaceFilter == "all" || namespaceFilter == ns.Name {
			nsList = append(nsList, ns.Name)
			g.Nodes = append(g.Nodes, GraphNode{
				ID:   "ns:" + ns.Name,
				Kind: "Namespace",
				Name: ns.Name,
			})
		}
	}

	// -------- Recursos por namespace --------
	for _, ns := range nsList {
		deps, _ := client.AppsV1().Deployments(ns).List(timeoutCtx, metav1.ListOptions{})
		stsList, _ := client.AppsV1().StatefulSets(ns).List(timeoutCtx, metav1.ListOptions{})
		dsList, _ := client.AppsV1().DaemonSets(ns).List(timeoutCtx, metav1.ListOptions{})
		rsList, _ := client.AppsV1().ReplicaSets(ns).List(timeoutCtx, metav1.ListOptions{})
		pods, _ := client.CoreV1().Pods(ns).List(timeoutCtx, metav1.ListOptions{})
		svcs, _ := client.CoreV1().Services(ns).List(timeoutCtx, metav1.ListOptions{})
		hpas, _ := client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(timeoutCtx, metav1.ListOptions{})

		for _, d := range deps.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "deploy:" + ns + ":" + d.Name,
				Kind:      "Deployment",
				Name:      d.Name,
				Namespace: ns,
				Labels:    d.Labels,
			})
		}

		for _, s := range stsList.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "sts:" + ns + ":" + s.Name,
				Kind:      "StatefulSet",
				Name:      s.Name,
				Namespace: ns,
				Labels:    s.Labels,
			})
		}

		for _, d := range dsList.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "ds:" + ns + ":" + d.Name,
				Kind:      "DaemonSet",
				Name:      d.Name,
				Namespace: ns,
				Labels:    d.Labels,
			})
		}

		for _, rs := range rsList.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "rs:" + ns + ":" + rs.Name,
				Kind:      "ReplicaSet",
				Name:      rs.Name,
				Namespace: ns,
				Labels:    rs.Labels,
			})
		}

		for _, p := range pods.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "pod:" + ns + ":" + p.Name,
				Kind:      "Pod",
				Name:      p.Name,
				Namespace: ns,
				Labels:    p.Labels,
			})
		}

		for _, s := range svcs.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "svc:" + ns + ":" + s.Name,
				Kind:      "Service",
				Name:      s.Name,
				Namespace: ns,
				Labels:    s.Labels,
			})
		}

		for _, h := range hpas.Items {
			g.Nodes = append(g.Nodes, GraphNode{
				ID:        "hpa:" + ns + ":" + h.Name,
				Kind:      "HPA",
				Name:      h.Name,
				Namespace: ns,
				Labels:    h.Labels,
			})
		}

		buildEdgesForNamespace(g, ns, deps, stsList, dsList, rsList, pods, svcs, hpas)
	}

	log.Printf(
		"[TOPOLOGY] nodes=%d edges=%d",
		len(g.Nodes),
		len(g.Edges),
	)

	/*
	========================
	 ADAPTER → REACT FLOW
	========================
	*/

	rfNodes := []RFNode{}
	x := 0.0
	y := 0.0

	for _, n := range g.Nodes {
		rfNodes = append(rfNodes, RFNode{
			ID:   n.ID,
			Type: "default",
			Position: map[string]float64{
				"x": x,
				"y": y,
			},
			Data: map[string]interface{}{
				"label": n.Kind + ": " + n.Name,
			},
		})

		y += 120
		if y > 800 {
			y = 0
			x += 260
		}
	}

	rfEdges := []RFEdge{}
	for _, e := range g.Edges {
		rfEdges = append(rfEdges, RFEdge{
			ID:     e.ID,
			Source: e.Source,
			Target: e.Target,
		})
	}

	return &RFGraph{
		Nodes: rfNodes,
		Edges: rfEdges,
	}, nil
}

/*
========================
 HELPERS
========================
*/

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

	// HPA -> Workload
	for _, h := range hpas.Items {
		ref := h.Spec.ScaleTargetRef
		switch ref.Kind {
		case "Deployment":
			g.Edges = append(g.Edges, GraphEdge{
				ID:     "edge:hpa->deploy:" + ns + ":" + h.Name + "->" + ref.Name,
				Source: "hpa:" + ns + ":" + h.Name,
				Target: "deploy:" + ns + ":" + ref.Name,
			})
		case "StatefulSet":
			g.Edges = append(g.Edges, GraphEdge{
				ID:     "edge:hpa->sts:" + ns + ":" + h.Name + "->" + ref.Name,
				Source: "hpa:" + ns + ":" + h.Name,
				Target: "sts:" + ns + ":" + ref.Name,
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
