import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiClient } from '../services/api';
import { useAuth } from '../auth/AuthContext';

type Cluster = {
  id: number;
  name: string;
  description: string;
};

const ClustersPage: React.FC = () => {
  const { username, role, logout } = useAuth();
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [kubeconfig, setKubeconfig] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const navigate = useNavigate();

  const fetchClusters = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<Cluster[]>('/clusters');
      setClusters(res.data);
    } catch (err) {
      setError('Erro ao carregar clusters');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchClusters();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!kubeconfig) {
      setError('Selecione um arquivo de kubeconfig');
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const text = await kubeconfig.text();
      const base64 = btoa(text);
      await apiClient.post('/clusters', {
        name,
        description,
        kubeconfigBase64: base64
      });
      setName('');
      setDescription('');
      setKubeconfig(null);
      await fetchClusters();
    } catch {
      setError('Erro ao criar cluster');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-950">
      <header className="flex items-center justify-between px-6 py-3 border-b border-slate-800 bg-slate-900/80">
        <div>
          <h1 className="text-lg font-semibold text-slate-50">Clusters Kubernetes</h1>
          <p className="text-xs text-slate-400">Visualização interativa de topologia</p>
        </div>
        <div className="flex items-center gap-4">
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

      <main className="flex-1 flex flex-col lg:flex-row gap-6 p-6">
        <section className="flex-1 bg-slate-900/70 border border-slate-800 rounded-xl p-4 overflow-auto">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-slate-200">Meus clusters</h2>
            {loading && <span className="text-xs text-slate-400">Carregando...</span>}
          </div>
          {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
          {clusters.length === 0 && !loading && (
            <p className="text-sm text-slate-400">Nenhum cluster cadastrado ainda.</p>
          )}
          <ul className="space-y-2">
            {clusters.map(c => (
              <li
                key={c.id}
                className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-900/80 px-3 py-2 hover:border-sky-500/60 cursor-pointer"
                onClick={() => navigate(`/topology/${c.id}`)}
              >
                <div>
                  <p className="text-sm text-slate-50">{c.name}</p>
                  {c.description && <p className="text-xs text-slate-400">{c.description}</p>}
                </div>
                <span className="text-[10px] text-slate-500">Ver topologia</span>
              </li>
            ))}
          </ul>
        </section>

        {role === 'admin' && (
          <section className="w-full lg:w-96 bg-slate-900/70 border border-slate-800 rounded-xl p-4">
            <h2 className="text-sm font-semibold text-slate-200 mb-4">Novo cluster</h2>
            <form className="space-y-3" onSubmit={handleCreate}>
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">Nome</label>
                <input
                  className="w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 text-sm text-slate-50 focus:outline-none focus:ring-2 focus:ring-sky-500"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">Descrição</label>
                <textarea
                  className="w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 text-sm text-slate-50 focus:outline-none focus:ring-2 focus:ring-sky-500"
                  rows={3}
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">Kubeconfig</label>
                <input
                  type="file"
                  accept=".yaml,.yml"
                  className="w-full text-xs text-slate-300 file:mr-3 file:rounded-md file:border-0 file:bg-sky-600 file:px-3 file:py-1 file:text-xs file:font-medium file:text-slate-50 hover:file:bg-sky-500"
                  onChange={e => setKubeconfig(e.target.files?.[0] ?? null)}
                />
              </div>
              <button
                type="submit"
                disabled={submitting}
                className="w-full rounded-lg bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 disabled:cursor-not-allowed py-2 text-sm font-semibold text-slate-50"
              >
                {submitting ? 'Salvando...' : 'Adicionar cluster'}
              </button>
            </form>
          </section>
        )}
      </main>
    </div>
  );
};

export default ClustersPage;

