import React, { useEffect, useState, useCallback, useMemo, useRef } from 'react';
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
import dagre from 'dagre';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import Ansi from 'ansi-to-react';

import 'reactflow/dist/style.css'; 
import { apiClient } from '../services/api';
import { useAuth } from '../auth/AuthContext';

// --- CONFIGURAÇÃO DE CORES ---
const resourceColors: Record<string, { bg: string; border: string; text: string }> = {
  Pod:           { bg: '#022c22', border: '#10b981', text: '#34d399' },
  Deployment:    { bg: '#0f172a', border: '#3b82f6', text: '#60a5fa' },
  Service:       { bg: '#2a1205', border: '#f97316', text: '#fb923c' },
  StatefulSet:   { bg: '#1e1b4b', border: '#8b5cf6', text: '#a78bfa' },
  DaemonSet:     { bg: '#370b18', border: '#ec4899', text: '#f472b6' },
  ReplicaSet:    { bg: '#171717', border: '#525252', text: '#a3a3a3' },
  HPA:           { bg: '#2e1065', border: '#c084fc', text: '#e879f9' },
  Node:          { bg: '#172554', border: '#1d4ed8', text: '#bfdbfe' },
  default:       { bg: '#020617', border: '#334155', text: '#e2e8f0' },
};

const getStatusColor = (status: string) => {
    if (!status) return 'transparent';
    if (status === 'Running' || status === 'Succeeded' || status === 'Ready') return '#10b981';
    if (status === 'Pending' || status === 'ContainerCreating' || status === 'Warning') return '#f59e0b';
    if (status === 'Failed' || status === 'Error' || status.includes('Crash') || status.includes('Image')) return '#ef4444';
    return '#6366f1';
};

// --- LAYOUT ENGINE (DAGRE - PAISAGEM FORÇADA) ---
const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));

    // 'LR' = Left to Right (Paisagem)
    // ranksep = Distância entre colunas
    // nodesep = Distância entre nós na mesma coluna
    dagreGraph.setGraph({ rankdir: 'LR', ranksep: 180, nodesep: 60 });

    nodes.forEach((node) => {
        // Largura/Altura do Card
        dagreGraph.setNode(node.id, { width: 220, height: 80 });
    });

    edges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = nodes.map((node) => {
        const nodeWithPosition = dagreGraph.node(node.id);
        
        // Fallback de segurança
        if (!nodeWithPosition) return node;

        return {
            ...node,
            // --- AQUI ESTÁ A CORREÇÃO CRÍTICA ---
            // Forçamos o React Flow a usar conectores laterais, alinhando com o 'LR' do Dagre
            targetPosition: Position.Left,
            sourcePosition: Position.Right,
            position: {
                x: nodeWithPosition.x - 110, // Ajuste para centralizar (metade da largura)
                y: nodeWithPosition.y - 40,  // Ajuste para centralizar (metade da altura)
            },
        };
    });

    return { nodes: layoutedNodes, edges };
};

// --- HELPER PARSERS ---
const parseNodeId = (id: string) => {
  const parts = id.split(':');
  if (parts.length === 3) return { kind: parts[0], namespace: parts[1], name: parts[2] };
  if (parts.length === 2) return { kind: parts[0], namespace: '', name: parts[1] };
  return { kind: 'unknown', namespace: '', name: id };
};

// --- COMPONENTES AUXILIARES ---

const YamlViewer = ({ clusterId, nodeId }: { clusterId: string, nodeId: string }) => {
  const [content, setContent] = useState('Carregando YAML...');
  useEffect(() => {
    const { kind, namespace, name } = parseNodeId(nodeId);
    const kindCap = kind.charAt(0).toUpperCase() + kind.slice(1);
    apiClient.get(`/clusters/${clusterId}/resources/yaml`, { params: { kind: kindCap, namespace, name } })
    .then(res => setContent(res.data))
    .catch(err => setContent(`Erro: ${err.message}`));
  }, [clusterId, nodeId]);

  return (
    <div className="h-full overflow-auto text-[10px] bg-[#020617]">
        <SyntaxHighlighter language="yaml" style={vscDarkPlus} customStyle={{ margin: 0, height: '100%', background: 'transparent' }}>
            {content}
        </SyntaxHighlighter>
    </div>
  );
};

const LogViewer = ({ clusterId, nodeId }: { clusterId: string, nodeId: string }) => {
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState('Conectando...');
  const [follow, setFollow] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const { kind, namespace, name } = parseNodeId(nodeId);
    if (kind.toLowerCase() !== 'pod') {
      setLogs(['Logs apenas disponíveis para Pods.']);
      return;
    }
    const fetchLogs = () => {
      apiClient.get(`/clusters/${clusterId}/resources/logs`, { params: { namespace, name, tail: 200 } })
      .then(res => {
        setLogs(res.data.lines || []);
        setStatus('Sync: ' + new Date().toLocaleTimeString());
      })
      .catch(() => setStatus('Erro na conexão'));
    };
    fetchLogs();
    const interval = setInterval(fetchLogs, 3000);
    return () => clearInterval(interval);
  }, [clusterId, nodeId]);

  // Auto-scroll
  useEffect(() => {
    if (follow && bottomRef.current) {
        bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, follow]);

  return (
    <div className="flex flex-col h-full bg-[#0d1117] rounded border border-slate-800 font-mono">
      {/* Toolbar */}
      <div className="flex justify-between items-center px-3 py-1 bg-slate-900 border-b border-slate-800">
        <span className="text-[10px] text-slate-400">Terminal Output</span>
        <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={follow} onChange={e => setFollow(e.target.checked)} className="rounded bg-slate-800 border-slate-600 text-sky-500 focus:ring-0 w-3 h-3" />
            <span className={`text-[10px] ${follow ? 'text-sky-400 font-bold' : 'text-slate-500'}`}>Auto-Follow</span>
        </label>
      </div>
      
      {/* Log Area */}
      <div className="flex-1 overflow-y-auto p-3 text-[10px] leading-relaxed text-slate-300">
        {logs.map((line, i) => (
          <div key={i} className="whitespace-pre-wrap break-all border-b border-slate-800/30 pb-0.5 mb-0.5 hover:bg-slate-800/50">
             <Ansi>{line}</Ansi>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
      <div className="text-[9px] text-slate-500 text-right p-1 bg-slate-950">{status}</div>
    </div>
  );
};

// --- NÓ CUSTOMIZADO (HOVER + CORES) ---
const ResourceNode = ({ data }: any) => {
  const isGroup = data.isGroup;
  
  if (isGroup) {
    return (
      <div style={{ 
        background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', 
        border: '2px solid #6366f1', borderRadius: '12px', padding: '20px', 
        width: '280px', height: '140px', color: 'white', display: 'flex', 
        flexDirection: 'column', alignItems: 'center', justifyContent: 'center', 
        boxShadow: '0 10px 25px -5px rgba(99, 102, 241, 0.4)', cursor: 'pointer'
      }} className="hover:scale-105 transition-transform group relative">
        <div style={{ fontSize: '18px', fontWeight: 'bold', marginBottom: '8px' }}>{data.label}</div>
        <div style={{ fontSize: '12px', color: '#a5b4fc', background: '#312e81', padding: '4px 10px', borderRadius: '20px' }}>Clique para entrar</div>
        <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
      </div>
    );
  }

  const kind = data.kind || 'default';
  const style = resourceColors[kind] || resourceColors.default;
  const statusColor = getStatusColor(data.status);

  return (
    <div className="group relative">
        {/* Card Principal */}
        <div style={{ 
            background: style.bg, 
            border: `1px solid ${data.status && data.status !== 'Running' && data.status !== 'Ready' ? statusColor : style.border}`, 
            borderRadius: '6px', padding: '6px 10px', width: '200px', 
            color: '#e2e8f0', textAlign: 'center', fontSize: '11px', 
            boxShadow: `0 4px 6px -1px rgba(0,0,0,0.5)`,
            transition: 'all 0.2s'
        }} className="hover:brightness-125 hover:shadow-lg hover:shadow-sky-900/20">
        
        {/* Status Dot */}
        {data.status && (
            <div style={{
                position: 'absolute', top: '-3px', right: '-3px', width: '8px', height: '8px',
                borderRadius: '50%', backgroundColor: statusColor, border: '1px solid #0f172a',
                boxShadow: statusColor === '#ef4444' ? '0 0 5px #ef4444' : 'none'
            }} />
        )}

        <Handle type="target" position={Position.Left} style={{ background: style.border, width: '6px', height: '6px', border: 'none' }} />
        
        <div style={{ fontWeight: '700', marginBottom: '2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: style.text }}>
            {data.label.includes(':') ? data.label.split(':')[1] : data.label}
        </div>
        
        <div className="flex justify-center gap-2 items-center mt-1">
            <span style={{ fontSize: '9px', color: style.text, background: 'rgba(0,0,0,0.3)', padding: '1px 5px', borderRadius: '3px', fontFamily: 'monospace' }}>
                {kind.toUpperCase()}
            </span>
            {data.status && data.status !== 'Running' && (
                <span style={{ fontSize: '8px', color: statusColor, fontWeight: 'bold' }}>{data.status}</span>
            )}
        </div>

        <Handle type="source" position={Position.Right} style={{ background: style.border, width: '6px', height: '6px', border: 'none' }} />
        </div>

        {/* TOOLTIP FLUTUANTE (HOVER) */}
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-48 bg-slate-900 border border-slate-700 rounded-md shadow-xl p-3 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-50">
            <div className="text-[10px] text-slate-400 mb-1">Detalhes do Recurso</div>
            <div className="text-xs font-bold text-white mb-2 break-all">{data.label}</div>
            <div className="grid grid-cols-2 gap-2 text-[10px]">
                <div>
                    <span className="block text-slate-500">Status</span>
                    <span style={{ color: statusColor }}>{data.status || 'Unknown'}</span>
                </div>
                <div>
                    <span className="block text-slate-500">Namespace</span>
                    <span className="text-slate-300">{data.namespace}</span>
                </div>
                <div>
                    <span className="block text-slate-500">Kind</span>
                    <span className="text-slate-300">{kind}</span>
                </div>
            </div>
            {/* Seta do tooltip */}
            <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-1 border-4 border-transparent border-t-slate-700"></div>
        </div>
    </div>
  );
};

// --- Tipagens ---
type GraphNode = {
  id: string; type: string; position: { x: number; y: number };
  data: { label: string; namespace?: string; kind?: string; status?: string; labels?: Record<string, string>; isGroup?: boolean; count?: number; originalNamespace?: string; id_short?: string; };
};
type GraphEdge = { id: string; source: string; target: string; };
type ClusterGraph = { nodes: GraphNode[]; edges: GraphEdge[]; };

const POLL_INTERVAL_MS = 15000;

// --- Componente React Flow ---
const TopologyContent: React.FC<{
  nodes: Node[]; edges: Edge[]; onNodesChange: any; onEdgesChange: any; onNodeClick: any; nodeTypes: any;
}> = ({ nodes, edges, onNodesChange, onEdgesChange, onNodeClick, nodeTypes }) => {
  const { fitView } = useReactFlow();
  
  // Fit View inicial suave
  useEffect(() => {
    if (nodes.length > 0) setTimeout(() => fitView({ padding: 0.1, duration: 800 }), 100);
  }, [nodes.length, fitView]);

  return (
    <ReactFlow
      nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onNodeClick={onNodeClick} nodeTypes={nodeTypes}
      minZoom={0.05} maxZoom={2} onlyRenderVisibleElements={true} defaultEdgeOptions={{ type: 'smoothstep', animated: true, style: { stroke: '#475569', strokeWidth: 1.5 } }}
      style={{ width: '100%', height: '100%', background: '#0f172a' }}
    >
      <MiniMap 
        nodeColor={(n) => n.data.isGroup ? '#6366f1' : (resourceColors[n.data.kind || 'default']?.border || '#334155')} 
        maskColor="#020617ee" style={{ backgroundColor: '#0f172a' }} 
      />
      <Controls style={{ backgroundColor: '#1e293b', border: '1px solid #334155' }} />
      <Background color="#1e293b" gap={30} size={1} />
    </ReactFlow>
  );
};

// --- PÁGINA PRINCIPAL ---
const TopologyPage: React.FC = () => {
  const { clusterId } = useParams<{ clusterId: string }>();
  const navigate = useNavigate();
  const { logout } = useAuth();

  const [fullGraph, setFullGraph] = useState<ClusterGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [expandedNamespace, setExpandedNamespace] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [activeTab, setActiveTab] = useState<'info' | 'yaml' | 'logs'>('info');
  const [searchTerm, setSearchTerm] = useState('');

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const nodeTypes = useMemo(() => ({ custom: ResourceNode }), []);

  const processView = useCallback(() => {
    if (!fullGraph || !fullGraph.nodes) return;

    let currentNodes = fullGraph.nodes;
    let currentEdges = fullGraph.edges;
    
    // VISÃO DE GRUPOS
    if (expandedNamespace === null) {
        const groups: Record<string, number> = {};
        fullGraph.nodes.forEach(n => {
            const ns = n.data.namespace || '_global_';
            groups[ns] = (groups[ns] || 0) + 1;
        });

        const groupNodes = Object.entries(groups)
            .filter(([ns]) => ns.toLowerCase().includes(searchTerm.toLowerCase()))
            .map(([ns, count], idx) => ({
                id: `ns-${ns}`, type: 'custom',
                data: { label: ns, count, isGroup: true, originalNamespace: ns },
                // Layout de Grid simples para grupos
                position: { x: (idx % 4) * 320, y: Math.floor(idx / 4) * 180 }
            }));
        
        setNodes(groupNodes);
        setEdges([]);
        return;
    }

    // VISÃO DE DETALHE
    currentNodes = currentNodes.filter(n => (n.data.namespace || '_global_') === expandedNamespace);
    
    // Forçar o tipo 'custom' para aplicar as cores e tooltip
    currentNodes = currentNodes.map(node => ({
        ...node,
        type: 'custom',
        data: {
             ...node.data,
             status: node.data.status || 'Unknown'
        }
    }));

    if (searchTerm) {
        currentNodes = currentNodes.filter(n => n.data.label.toLowerCase().includes(searchTerm.toLowerCase()));
    }

    const nodeIds = new Set(currentNodes.map(n => n.id));
    currentEdges = fullGraph.edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));

    // LAYOUT DAGRE (Horizontal)
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(currentNodes, currentEdges);

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);

  }, [fullGraph, expandedNamespace, searchTerm, setNodes, setEdges]);

  useEffect(() => { processView(); }, [processView]);

  const fetchGraph = async (isBackground = false) => {
    if (!clusterId) return;
    if (!isBackground) setLoading(true);
    try {
      const res = await apiClient.get<ClusterGraph>(`/topology/${clusterId}`, { params: { namespace: 'all' } });
      if (res.data?.nodes) setFullGraph(res.data);
    } catch (err) { console.error(err); } 
    finally { if (!isBackground) setLoading(false); }
  };

  useEffect(() => { void fetchGraph(false); }, [clusterId]);
  useEffect(() => { const id = setInterval(() => void fetchGraph(true), POLL_INTERVAL_MS); return () => clearInterval(id); }, [clusterId]);

  const handleNodeClick = (_: React.MouseEvent, node: Node) => {
    if (node.data.isGroup) {
      setExpandedNamespace(node.data.originalNamespace);
      setSelectedNode(null);
      setSearchTerm('');
    } else {
      const originalNode = fullGraph?.nodes.find(n => n.id === node.id);
      if (originalNode) { setSelectedNode(originalNode); setActiveTab('info'); }
    }
  };

  return (
    <div className="h-screen w-screen bg-slate-950 flex flex-col overflow-hidden relative">
      <header className="h-14 flex-none flex items-center justify-between px-6 border-b border-slate-800 bg-slate-900 z-50 shadow-md">
        <div className="flex items-center gap-4 flex-1">
          <button onClick={() => navigate('/clusters')} className="text-xs px-3 py-1.5 rounded-full border border-slate-600 text-slate-300 hover:bg-slate-800">&larr; Voltar</button>
          <div className="h-5 w-px bg-slate-700 mx-1" />
          <h1 className="text-sm font-medium text-slate-200 flex items-center gap-2 whitespace-nowrap">
            {expandedNamespace ? (
               <> 
                 <span onClick={() => { setExpandedNamespace(null); setSelectedNode(null); }} className="cursor-pointer text-slate-400 hover:text-white">Namespaces</span> 
                 <span className="text-slate-600">/</span> 
                 <span className="text-sky-400 font-semibold">{expandedNamespace}</span> 
               </> 
            ) : 'Visão Geral'}
          </h1>
          <div className="ml-8 relative max-w-md w-full">
              <input type="text" placeholder="Buscar recursos..." value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-full py-1.5 px-4 text-xs text-white focus:outline-none focus:border-sky-500 transition-colors" />
          </div>
        </div>
        <button onClick={logout} className="text-xs text-slate-400 hover:text-white">Sair</button>
      </header>

      <main className="flex-1 relative w-full">
        {loading && <div className="absolute inset-0 z-50 flex items-center justify-center bg-slate-950/60 backdrop-blur-sm"><span className="text-sky-400 animate-pulse font-mono">Carregando Topologia...</span></div>}
        
        <ReactFlowProvider>
            <TopologyContent nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onNodeClick={handleNodeClick} nodeTypes={nodeTypes} />
        </ReactFlowProvider>

        <aside className={`absolute right-0 top-0 bottom-0 w-[600px] bg-slate-900 border-l border-slate-800 shadow-2xl backdrop-blur-sm transition-transform duration-300 ease-in-out flex flex-col z-40 ${selectedNode ? 'translate-x-0' : 'translate-x-full'}`}>
            {selectedNode && (
                <>
                    <div className="flex-none p-4 border-b border-slate-800 bg-slate-950/50">
                        <div className="flex justify-between items-start mb-2">
                            <h2 className="text-xs font-bold uppercase tracking-wider text-slate-500">Detalhes</h2>
                            <button onClick={() => setSelectedNode(null)} className="text-slate-400 hover:text-white"><svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg></button>
                        </div>
                        <div className="font-semibold text-slate-200 text-lg">{selectedNode.data.label}</div>
                        <div className="flex gap-2 mt-2">
                            <span className="text-[10px] bg-slate-800 px-2 py-0.5 rounded text-slate-400 border border-slate-700">{selectedNode.data.kind}</span>
                            {selectedNode.data.status && <span className="text-[10px] px-2 py-0.5 rounded text-white font-bold" style={{ backgroundColor: getStatusColor(selectedNode.data.status) }}>{selectedNode.data.status}</span>}
                        </div>
                    </div>
                    <div className="flex border-b border-slate-800 bg-slate-900">
                        {['info', 'yaml', 'logs'].map(tab => (
                            <button key={tab} onClick={() => setActiveTab(tab as any)} className={`flex-1 py-3 text-xs font-bold uppercase tracking-wide transition-colors ${activeTab === tab ? 'text-sky-400 border-b-2 border-sky-400 bg-slate-800/50' : 'text-slate-500 hover:text-slate-200'}`}>{tab}</button>
                        ))}
                    </div>
                    <div className="flex-1 overflow-hidden relative bg-[#020617]">
                        {activeTab === 'info' && (
                            <div className="absolute inset-0 overflow-y-auto p-5 space-y-6">
                                <div><label className="text-[10px] text-slate-500 uppercase font-bold block mb-1">ID</label><div className="font-mono text-xs text-slate-300 bg-slate-950 p-2 rounded border border-slate-800 break-all">{selectedNode.id}</div></div>
                                {selectedNode.data.labels && (
                                    <div><label className="text-[10px] text-slate-500 uppercase font-bold block mb-2">Labels</label>
                                        <div className="flex flex-wrap gap-2">{Object.entries(selectedNode.data.labels).map(([k, v]) => (
                                            <div key={k} className="flex text-[10px] border border-slate-700 rounded overflow-hidden"><span className="bg-slate-800 text-slate-400 px-2 py-1">{k}</span><span className="bg-slate-950 text-slate-200 px-2 py-1 border-l border-slate-700">{v as string}</span></div>
                                        ))}</div>
                                    </div>
                                )}
                            </div>
                        )}
                        {activeTab === 'yaml' && <div className="absolute inset-0 p-0"><YamlViewer clusterId={clusterId || ''} nodeId={selectedNode.id} /></div>}
                        {activeTab === 'logs' && <div className="absolute inset-0 p-0"><LogViewer clusterId={clusterId || ''} nodeId={selectedNode.id} /></div>}
                    </div>
                </>
            )}
        </aside>
      </main>
    </div>
  );
};

export default TopologyPage;
