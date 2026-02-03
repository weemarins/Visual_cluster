import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  useNodesState,
  useEdgesState,
  Position,
  useReactFlow,
  ReactFlowProvider,
  Handle
} from 'reactflow';

import 'reactflow/dist/style.css'; 
import { apiClient } from '../services/api';
import { useAuth } from '../auth/AuthContext';

// --- 1. COMPONENTES AUXILIARES DA SIDEBAR ---

// Helper para extrair dados do ID (ex: "pod:default:nginx-123" -> kind, ns, name)
const parseNodeId = (id: string) => {
  const parts = id.split(':');
  // Formato esperado: kind:namespace:name (ex: pod:default:app)
  if (parts.length === 3) {
    return { kind: parts[0], namespace: parts[1], name: parts[2] };
  }
  // Formato global: kind:name (ex: node:worker-1)
  if (parts.length === 2) {
    return { kind: parts[0], namespace: '', name: parts[1] };
  }
  return { kind: 'unknown', namespace: '', name: id };
};

const YamlViewer = ({ clusterId, nodeId }: { clusterId: string, nodeId: string }) => {
  const [content, setContent] = useState('Carregando YAML...');
  
  useEffect(() => {
    const { kind, namespace, name } = parseNodeId(nodeId);
    
    // Capitaliza o Kind (pod -> Pod) para o K8s entender, se necessário
    const kindCap = kind.charAt(0).toUpperCase() + kind.slice(1);

    apiClient.get(`/clusters/${clusterId}/resources/yaml`, {
      params: { kind: kindCap, namespace, name }
    })
    .then(res => setContent(res.data))
    .catch(err => {
      console.error(err);
      setContent(`Erro ao carregar YAML.\n${err.response?.data?.error || err.message}`);
    });
  }, [clusterId, nodeId]);

  return (
    <pre className="text-[10px] font-mono text-green-300 bg-slate-950 p-3 rounded border border-slate-800 overflow-auto h-full whitespace-pre-wrap">
      {content}
    </pre>
  );
};

const LogViewer = ({ clusterId, nodeId }: { clusterId: string, nodeId: string }) => {
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState('Conectando...');

  useEffect(() => {
    const { kind, namespace, name } = parseNodeId(nodeId);

    // Logs só fazem sentido para Pods (geralmente)
    if (kind.toLowerCase() !== 'pod') {
      setLogs(['Logs estão disponíveis apenas para Pods.']);
      setStatus('');
      return;
    }

    setLogs([]);
    setStatus('Buscando logs...');

    const fetchLogs = () => {
      apiClient.get(`/clusters/${clusterId}/resources/logs`, {
        params: { namespace, name, tail: 50 }
      })
      .then(res => {
        setLogs(res.data.lines || []);
        setStatus('Atualizado em ' + new Date().toLocaleTimeString());
      })
      .catch(err => {
        setStatus('Erro ao buscar logs.');
        setLogs([err.response?.data?.error || 'Falha na conexão']);
      });
    };

    fetchLogs();
    const interval = setInterval(fetchLogs, 4000); // Atualiza a cada 4s
    return () => clearInterval(interval);

  }, [clusterId, nodeId]);

  return (
    <div className="flex flex-col h-full">
      <div className="bg-black text-slate-300 font-mono text-[10px] p-3 flex-1 overflow-y-auto rounded border border-slate-800">
        {logs.length === 0 && <span className="text-slate-500 italic">Nenhum log encontrado ou carregando...</span>}
        {logs.map((line, i) => (
          <div key={i} className="border-b border-slate-900/50 py-0.5 break-all hover:bg-slate-900">{line}</div>
        ))}
      </div>
      <div className="text-[9px] text-slate-500 text-right mt-1">{status}</div>
    </div>
  );
};

// --- 2. NÓ CUSTOMIZADO ---
const ResourceNode = ({ data }: any) => {
  const isGroup = data.isGroup;
  
  if (isGroup) {
    return (
      <div style={{ 
        background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', 
        border: '2px solid #6366f1', borderRadius: '12px', padding: '20px', 
        width: '280px', height: '140px', color: 'white', display: 'flex', 
        flexDirection: 'column', alignItems: 'center', justifyContent: 'center', 
        boxShadow: '0 10px 25px -5px rgba(99, 102, 241, 0.4)', cursor: 'pointer',
        transition: 'transform 0.2s'
      }}>
        <div style={{ fontSize: '18px', fontWeight: 'bold', marginBottom: '8px' }}>{data.label}</div>
        <div style={{ fontSize: '12px', color: '#a5b4fc', background: '#312e81', padding: '4px 10px', borderRadius: '20px' }}>
            Clique para entrar
        </div>
        <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
        <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
      </div>
    );
  }

  return (
    <div style={{ 
      background: '#020617', border: '1px solid #334155', borderRadius: '6px', 
      padding: '8px', width: '160px', color: '#e2e8f0', textAlign: 'center', 
      fontSize: '11px', boxShadow: '0 2px 4px rgba(0,0,0,0.5)', position: 'relative'
    }}>
      <Handle type="target" position={Position.Left} style={{ background: '#38bdf8', width: '6px', height: '6px' }} />
      <div style={{ fontWeight: '600', marginBottom: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {data.label}
      </div>
      <div style={{ fontSize: '9px', color: '#64748b', fontFamily: 'monospace' }}>{data.id_short}</div>
      <Handle type="source" position={Position.Right} style={{ background: '#38bdf8', width: '6px', height: '6px' }} />
    </div>
  );
};

// --- Tipagens ---
type GraphNode = {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: {
    label: string;
    namespace?: string;
    labels?: Record<string, string>;
    isGroup?: boolean;
    count?: number;
    originalNamespace?: string;
    id_short?: string;
  };
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

// --- Componente React Flow Interno ---
const TopologyContent: React.FC<{
  nodes: Node[];
  edges: Edge[];
  onNodesChange: any;
  onEdgesChange: any;
  onNodeClick: any;
  nodeTypes: any;
}> = ({ nodes, edges, onNodesChange, onEdgesChange, onNodeClick, nodeTypes }) => {
  const { fitView } = useReactFlow();

  useEffect(() => {
    if (nodes.length > 0) {
        setTimeout(() => {
            fitView({ padding: 0.15, duration: 800 });
        }, 100);
    }
  }, [nodes.length, fitView]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={onNodeClick}
      nodeTypes={nodeTypes}
      minZoom={0.02}
      maxZoom={3}
      onlyRenderVisibleElements={true} 
      defaultEdgeOptions={{ type: 'smoothstep', animated: false }}
      style={{ width: '100%', height: '100%', background: '#0f172a' }}
    >
      <MiniMap 
        nodeColor={(n) => n.data.isGroup ? '#6366f1' : '#334155'} 
        maskColor="#020617ee" 
        style={{ backgroundColor: '#0f172a' }}
      />
      <Controls style={{ backgroundColor: '#1e293b', border: '1px solid #334155', padding: '2px' }} />
      <Background color="#1e293b" gap={40} size={1} />
    </ReactFlow>
  );
};

// --- Página Principal ---
const TopologyPage: React.FC = () => {
  const { clusterId } = useParams<{ clusterId: string }>();
  const navigate = useNavigate();
  const { logout } = useAuth();

  const [fullGraph, setFullGraph] = useState<ClusterGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [expandedNamespace, setExpandedNamespace] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [hasMultipleNamespaces, setHasMultipleNamespaces] = useState(true);
  
  // Estado para controlar as abas da Sidebar
  const [activeTab, setActiveTab] = useState<'info' | 'yaml' | 'logs'>('info');

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const nodeTypes = useMemo(() => ({ custom: ResourceNode }), []);

  const computeLayout = useCallback((nodesToLayout: any[], edgesToLayout: any[], isGroupMode: boolean) => {
    const count = nodesToLayout.length;
    if (count === 0) return { nodes: [], edges: [] };

    const COLS = Math.ceil(Math.sqrt(count * (isGroupMode ? 2 : 1.8))) || 4;
    const X_GAP = isGroupMode ? 320 : 200;
    const Y_GAP = isGroupMode ? 180 : 100;

    const layoutNodes = nodesToLayout.map((n, index) => {
      const col = index % COLS;
      const row = Math.floor(index / COLS);
      
      return {
        id: n.id,
        type: 'custom',
        data: { 
            ...n.data, 
            id_short: n.id.length > 25 ? n.id.substring(0, 22) + '...' : n.id 
        },
        position: { x: col * X_GAP, y: row * Y_GAP },
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
      };
    });

    return { nodes: layoutNodes, edges: edgesToLayout };
  }, []);

  const processView = useCallback(() => {
    if (!fullGraph || !fullGraph.nodes) return;

    const groups: Record<string, number> = {};
    fullGraph.nodes.forEach(n => {
      const ns = n.data.namespace || '_global_';
      groups[ns] = (groups[ns] || 0) + 1;
    });

    const uniqueNamespaces = Object.keys(groups);

    if (uniqueNamespaces.length === 1 && expandedNamespace === null) {
        setHasMultipleNamespaces(false);
        setExpandedNamespace(uniqueNamespaces[0]);
        return;
    }

    if (uniqueNamespaces.length > 1) setHasMultipleNamespaces(true);

    if (expandedNamespace === null) {
      const groupNodes = Object.entries(groups).map(([ns, count]) => ({
        id: `ns-${ns}`,
        type: 'custom',
        data: { 
            label: ns === '_global_' ? 'Recursos Globais' : ns, 
            count, 
            isGroup: true, 
            originalNamespace: ns 
        },
        position: { x: 0, y: 0 }
      }));
      
      const { nodes: lNodes } = computeLayout(groupNodes, [], true);
      setNodes(lNodes);
      setEdges([]); 
    
    } else {
      const filteredNodes = fullGraph.nodes.filter(n => {
        const ns = n.data.namespace || '_global_';
        return ns === expandedNamespace;
      });

      const nodeIds = new Set(filteredNodes.map(n => n.id));
      const filteredEdges = fullGraph.edges.filter(e => 
        nodeIds.has(e.source) && nodeIds.has(e.target)
      ).map(e => ({
          ...e,
          style: { stroke: '#38bdf8', opacity: 0.3 }
      }));

      const { nodes: lNodes, edges: lEdges } = computeLayout(filteredNodes, filteredEdges, false);
      setNodes(lNodes);
      setEdges(lEdges);
    }
  }, [fullGraph, expandedNamespace, computeLayout, setNodes, setEdges]);

  useEffect(() => {
    processView();
  }, [processView]);

  const fetchGraph = async (isBackground = false) => {
    if (!clusterId) return;
    if (!isBackground) setLoading(true);
    
    try {
      const res = await apiClient.get<ClusterGraph>(`/topology/${clusterId}`, {
        params: { namespace: 'all' }
      });
      if (res.data?.nodes) {
          setFullGraph(res.data);
      }
    } catch (err) {
      console.error(err);
    } finally {
      if (!isBackground) setLoading(false);
    }
  };

  useEffect(() => {
    void fetchGraph(false);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  useEffect(() => {
    const id = setInterval(() => void fetchGraph(true), POLL_INTERVAL_MS);
    return () => clearInterval(id);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  const handleNodeClick = (_: React.MouseEvent, node: Node) => {
    if (node.data.isGroup) {
      setExpandedNamespace(node.data.originalNamespace);
      setSelectedNode(null);
    } else {
      const originalNode = fullGraph?.nodes.find(n => n.id === node.id);
      if (originalNode) {
          setSelectedNode(originalNode);
          setActiveTab('info'); // Reseta a aba ao abrir novo nó
      }
    }
  };

  const handleBackToOverview = () => {
    setExpandedNamespace(null);
    setSelectedNode(null);
  };

  return (
    <div className="h-screen w-screen bg-slate-950 flex flex-col overflow-hidden relative">
      <header className="h-14 flex-none flex items-center justify-between px-6 border-b border-slate-800 bg-slate-900 z-50 shadow-md">
        <div className="flex items-center gap-4">
          <button onClick={() => navigate('/clusters')} className="text-xs px-3 py-1.5 rounded-full border border-slate-600 text-slate-300 hover:bg-slate-800 transition-colors">
            &larr; Voltar
          </button>
          
          <div className="h-5 w-px bg-slate-700 mx-1" />

          <h1 className="text-sm font-medium text-slate-200 flex items-center gap-2">
            {expandedNamespace ? (
               <>
                 {hasMultipleNamespaces && (
                     <>
                        <span onClick={handleBackToOverview} className="cursor-pointer text-slate-400 hover:text-white transition-colors">Namespaces</span>
                        <span className="text-slate-600">/</span>
                     </>
                 )}
                 <span className="text-sky-400 font-semibold">{expandedNamespace === '_global_' ? 'Recursos Globais' : expandedNamespace}</span>
                 <span className="ml-2 text-xs bg-slate-800 px-2 py-0.5 rounded text-slate-400">
                    {nodes.length} nós
                 </span>
               </>
            ) : 'Visão Geral dos Namespaces'}
          </h1>
        </div>
        <button onClick={logout} className="text-xs text-slate-400 hover:text-white">Sair</button>
      </header>

      <main className="flex-1 relative w-full h-full">
        <div className="absolute inset-0 z-0">
            {loading && (
                <div className="absolute inset-0 flex items-center justify-center z-50 bg-slate-950/60 backdrop-blur-sm">
                   <div className="flex items-center gap-3 bg-slate-900 px-6 py-3 rounded-full border border-sky-500/30 shadow-xl">
                      <span className="w-2 h-2 bg-sky-500 rounded-full animate-pulse"/>
                      <span className="text-sm text-sky-400">Carregando topologia...</span>
                   </div>
                </div>
            )}

            <ReactFlowProvider>
                <TopologyContent 
                    nodes={nodes} 
                    edges={edges} 
                    onNodesChange={onNodesChange} 
                    onEdgesChange={onEdgesChange}
                    onNodeClick={handleNodeClick}
                    nodeTypes={nodeTypes}
                />
            </ReactFlowProvider>
        </div>

        {/* --- SIDEBAR REFORMULADA --- */}
        <aside 
            className={`absolute right-0 top-0 bottom-0 w-96 bg-slate-900 border-l border-slate-800 shadow-2xl backdrop-blur-sm transition-transform duration-300 ease-in-out flex flex-col z-40 ${selectedNode ? 'translate-x-0' : 'translate-x-full'}`}
        >
            {selectedNode && (
                <>
                    {/* Cabeçalho Fixo */}
                    <div className="flex-none p-4 border-b border-slate-800 bg-slate-950/50">
                        <div className="flex justify-between items-start mb-2">
                            <h2 className="text-xs font-bold uppercase tracking-wider text-slate-500">Recurso</h2>
                            <button onClick={() => setSelectedNode(null)} className="text-slate-400 hover:text-white p-1 hover:bg-slate-800 rounded">
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                            </button>
                        </div>
                        <div className="font-semibold text-slate-200 break-all leading-tight">
                            {selectedNode.data.label}
                        </div>
                    </div>

                    {/* Abas */}
                    <div className="flex border-b border-slate-800 bg-slate-900">
                        <button 
                            onClick={() => setActiveTab('info')}
                            className={`flex-1 py-2.5 text-xs font-medium transition-colors ${activeTab === 'info' ? 'text-sky-400 border-b-2 border-sky-400 bg-slate-800/50' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-200'}`}
                        >
                            Info
                        </button>
                        <button 
                            onClick={() => setActiveTab('yaml')}
                            className={`flex-1 py-2.5 text-xs font-medium transition-colors ${activeTab === 'yaml' ? 'text-sky-400 border-b-2 border-sky-400 bg-slate-800/50' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-200'}`}
                        >
                            YAML
                        </button>
                        <button 
                            onClick={() => setActiveTab('logs')}
                            className={`flex-1 py-2.5 text-xs font-medium transition-colors ${activeTab === 'logs' ? 'text-sky-400 border-b-2 border-sky-400 bg-slate-800/50' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-200'}`}
                        >
                            Logs
                        </button>
                    </div>

                    {/* Conteúdo com Scroll */}
                    <div className="flex-1 overflow-hidden relative bg-slate-900">
                        {activeTab === 'info' && (
                            <div className="absolute inset-0 overflow-y-auto p-4 space-y-6">
                                <div>
                                    <label className="text-[10px] text-slate-500 uppercase font-bold block mb-1.5">ID Técnico</label>
                                    <div className="font-mono text-[11px] text-slate-400 break-all bg-slate-950/50 p-2 rounded border border-dashed border-slate-800">
                                        {selectedNode.id}
                                    </div>
                                </div>

                                {selectedNode.data.labels && Object.keys(selectedNode.data.labels).length > 0 && (
                                    <div>
                                        <label className="text-[10px] text-slate-500 uppercase font-bold block mb-2">Labels</label>
                                        <div className="flex flex-wrap gap-2">
                                            {Object.entries(selectedNode.data.labels).map(([k, v]) => (
                                                <div key={k} className="flex text-[10px] border border-slate-700 rounded overflow-hidden shadow-sm">
                                                    <span className="bg-slate-800 text-slate-400 px-2 py-1 font-medium">{k}</span>
                                                    <span className="bg-slate-900 text-slate-200 px-2 py-1 border-l border-slate-700">{v as string}</span>
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </div>
                        )}

                        {activeTab === 'yaml' && (
                            <div className="absolute inset-0 p-4">
                                <YamlViewer clusterId={clusterId || ''} nodeId={selectedNode.id} />
                            </div>
                        )}

                        {activeTab === 'logs' && (
                            <div className="absolute inset-0 p-4">
                                <LogViewer clusterId={clusterId || ''} nodeId={selectedNode.id} />
                            </div>
                        )}
                    </div>
                </>
            )}
        </aside>
      </main>
    </div>
  );
};

export default TopologyPage;