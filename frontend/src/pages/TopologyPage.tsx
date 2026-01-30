import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  useNodesState,
  useEdgesState,
  Position
} from 'reactflow';
import { apiClient } from '../services/api';
import { useAuth } from '../auth/AuthContext';

type GraphNode = {
  id: string;
  kind: string;
  name: string;
  namespace?: string;
  labels?: Record<string, string>;
};

type GraphEdge = {
  id: string;
  source: string;
  target: string;
};

type ClusterGraph = {
  nodes: GraphNode[];
  edges: GraphEdge[];
};

const POLL_INTERVAL_MS = 15000;

const TopologyPage: React.FC = () => {
  const { clusterId } = useParams<{ clusterId: string }>();
  const navigate = useNavigate();
  const { username, role, logout } = useAuth();

  const [namespaceFilter, setNamespaceFilter] = useState<string>('all');
  const [graph, setGraph] = useState<ClusterGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<GraphNode | null>(null);

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const namespaces = useMemo(() => {
    const set = new Set<string>();
    graph?.nodes.forEach(n => {
      if (n.namespace) set.add(n.namespace);
    });
    return ['all', ...Array.from(set).sort()];
  }, [graph]);

  const layout = (graph: ClusterGraph): { nodes: Node[]; edges: Edge[] } => {
    const levelOrder = ['Namespace', 'Deployment', 'StatefulSet', 'DaemonSet', 'ReplicaSet', 'Pod', 'Service', 'HPA', 'Node'];
    const kindIndex = (kind: string) => levelOrder.indexOf(kind) >= 0 ? levelOrder.indexOf(kind) : levelOrder.length;

    const grouped = new Map<number, GraphNode[]>();
    graph.nodes.forEach(n => {
      const level = kindIndex(n.kind);
      const arr = grouped.get(level) ?? [];
      arr.push(n);
      grouped.set(level, arr);
    });

    const rfNodes: Node[] = [];
    const rfEdges: Edge[] = graph.edges.map(e => ({
      id: e.id,
      source: e.source,
      target: e.target,
      animated: false,
      style: { stroke: '#38bdf8' }
    }));

    const xGap = 220;
    const yGap = 120;

    Array.from(grouped.entries())
      .sort((a, b) => a[0] - b[0])
      .forEach(([level, group]) => {
        group.forEach((n, idx) => {
          rfNodes.push({
            id: n.id,
            data: { label: `${n.kind}: ${n.name}` },
            position: { x: idx * xGap, y: level * yGap },
            sourcePosition: Position.Bottom,
            targetPosition: Position.Top,
            style: {
              padding: 8,
              borderRadius: 8,
              fontSize: 11,
              border: '1px solid #1e293b',
              background:
                n.kind === 'Pod'
                  ? '#0f172a'
                  : n.kind === 'Service'
                  ? '#0f172a'
                  : '#020617',
              color: '#e5e7eb'
            }
          });
        });
      });

    return { nodes: rfNodes, edges: rfEdges };
  };

  const fetchGraph = async () => {
    if (!clusterId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiClient.get<ClusterGraph>(`/topology/${clusterId}`, {
        params: { namespace: namespaceFilter }
      });
      setGraph(res.data);
      const { nodes, edges } = layout(res.data);
      setNodes(nodes);
      setEdges(edges);
    } catch {
      setError('Erro ao carregar topologia');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchGraph();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespaceFilter]);

  useEffect(() => {
    const id = window.setInterval(() => {
      void fetchGraph();
    }, POLL_INTERVAL_MS);
    return () => window.clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespaceFilter]);

  const onNodeClick = (_: React.MouseEvent, node: Node) => {
    const found = graph?.nodes.find(n => n.id === node.id) ?? null;
    setSelected(found);
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-950">
      <header className="flex items-center justify-between px-6 py-3 border-b border-slate-800 bg-slate-900/80">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/clusters')}
            className="text-xs px-3 py-1 rounded-full border border-slate-600 text-slate-200 hover:bg-slate-800"
          >
            Voltar
          </button>
          <div>
            <h1 className="text-lg font-semibold text-slate-50">Topologia do cluster #{clusterId}</h1>
            <p className="text-xs text-slate-400">Zoom, pan, seleção de recursos e painel lateral</p>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-300">Namespace</label>
            <select
              value={namespaceFilter}
              onChange={e => setNamespaceFilter(e.target.value)}
              className="rounded-md border border-slate-700 bg-slate-950/70 text-xs px-2 py-1 text-slate-100"
            >
              {namespaces.map(ns => (
                <option key={ns} value={ns}>
                  {ns === 'all' ? 'Todos' : ns}
                </option>
              ))}
            </select>
          </div>
          <div className="text-right">
            <p className="text-xs text-slate-400">Logado como</p>
            <p className="text-sm text-slate-50">
              {username} <span className="text-sky-400 text-xs">({role})</span>
            </p>
          </div>
          <button
            onClick={logout}
            className="text-xs px-3 py-1 rounded-full border border-slate-600 text-slate-200 hover:bg-slate-800"
          >
            Sair
          </button>
        </div>
      </header>

      <main className="flex-1 flex overflow-hidden">
        <section className="flex-1 relative">
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center z-10 pointer-events-none">
              <div className="rounded-full border border-sky-500/40 px-4 py-2 text-xs text-sky-300 bg-slate-950/80">
                Carregando topologia...
              </div>
            </div>
          )}
          {error && (
            <div className="absolute top-3 left-3 z-10 rounded-md bg-red-900/80 border border-red-700 px-3 py-2 text-xs text-red-100">
              {error}
            </div>
          )}
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            fitView
            fitViewOptions={{ padding: 0.2 }}
          >
            <MiniMap
              nodeColor={() => '#0f172a'}
              nodeStrokeColor={() => '#38bdf8'}
              maskColor="#020617dd"
            />
            <Controls />
            <Background />
          </ReactFlow>
        </section>
        <aside className="w-80 border-l border-slate-800 bg-slate-900/80 p-4 hidden md:block">
          <h2 className="text-sm font-semibold text-slate-200 mb-3">Detalhes do recurso</h2>
          {!selected && <p className="text-xs text-slate-500">Selecione um nó no grafo.</p>}
          {selected && (
            <div className="space-y-2 text-xs text-slate-200">
              <div>
                <p className="text-[10px] uppercase tracking-wide text-slate-400">Tipo</p>
                <p className="text-sm">{selected.kind}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-slate-400">Nome</p>
                <p className="text-sm break-all">{selected.name}</p>
              </div>
              {selected.namespace && (
                <div>
                  <p className="text-[10px] uppercase tracking-wide text-slate-400">Namespace</p>
                  <p className="text-sm">{selected.namespace}</p>
                </div>
              )}
              {selected.labels && Object.keys(selected.labels).length > 0 && (
                <div>
                  <p className="text-[10px] uppercase tracking-wide text-slate-400 mb-1">Labels</p>
                  <div className="flex flex-wrap gap-1">
                    {Object.entries(selected.labels).map(([k, v]) => (
                      <span
                        key={k}
                        className="rounded-full border border-slate-700 bg-slate-950/60 px-2 py-0.5 text-[10px] text-slate-200"
                      >
                        {k}={v}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </aside>
      </main>
    </div>
  );
};

export default TopologyPage;

