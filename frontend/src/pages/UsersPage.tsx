import React, { useEffect, useState } from 'react';
import { apiClient } from '../services/api';
import { useAuth } from '../auth/AuthContext';
import { useNavigate } from 'react-router-dom';

type User = { username: string; displayName?: string; role?: string };

const UsersPage: React.FC = () => {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState('viewer');

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<User[]>('/auth/users');
      setUsers(res.data);
    } catch (err: any) {
      setError(err?.message || 'Erro ao carregar usuários');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void fetchUsers(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      await apiClient.post('/auth/users', { username, displayName, password, role });
      setUsername(''); setDisplayName(''); setPassword(''); setRole('viewer');
      await fetchUsers();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Erro ao criar usuário');
    }
  };

  const handleReset = async (user: string) => {
    const pw = window.prompt(`Nova senha para ${user}:`);
    if (!pw) return;
    try {
      await apiClient.post(`/auth/users/${encodeURIComponent(user)}/password`, { password: pw });
      alert('Senha atualizada');
    } catch (err: any) {
      alert(err?.response?.data?.error || 'Erro ao atualizar senha');
    }
  };

  const handleDelete = async (user: string) => {
    if (!window.confirm(`Remover usuário ${user}?`)) return;
    try {
      await apiClient.delete(`/auth/users/${encodeURIComponent(user)}`);
      await fetchUsers();
    } catch (err: any) {
      alert(err?.response?.data?.error || 'Erro ao deletar usuário');
    }
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-950 p-6">
      <header className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-semibold text-slate-50">Gerenciamento de Acessos</h1>
          <p className="text-xs text-slate-400">Apenas administradores podem acessar esta página</p>
        </div>
        <div>
          <button onClick={() => navigate('/clusters')} className="text-sm text-slate-300 hover:underline">Voltar</button>
        </div>
      </header>

      <main className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <section className="col-span-2 bg-slate-900/70 border border-slate-800 rounded-xl p-4">
          <h2 className="text-sm font-semibold text-slate-200 mb-3">Usuários</h2>
          {loading && <p className="text-sm text-slate-400">Carregando...</p>}
          {error && <p className="text-sm text-red-400">{error}</p>}
          <ul className="space-y-2">
            {users.map(u => (
              <li key={u.username} className="flex items-center justify-between p-2 border border-slate-800 rounded">
                <div>
                  <div className="text-sm text-slate-50">{u.username} <span className="text-xs text-slate-400">({u.role})</span></div>
                  <div className="text-xs text-slate-400">{u.displayName}</div>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => handleReset(u.username)} className="text-xs px-2 py-1 bg-amber-600 rounded text-white">Resetar</button>
                  <button onClick={() => handleDelete(u.username)} className="text-xs px-2 py-1 bg-red-600 rounded text-white">Excluir</button>
                </div>
              </li>
            ))}
          </ul>
        </section>

        <section className="bg-slate-900/70 border border-slate-800 rounded-xl p-4">
          <h2 className="text-sm font-semibold text-slate-200 mb-3">Criar usuário</h2>
          <form className="space-y-3" onSubmit={handleCreate}>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Usuário</label>
              <input value={username} onChange={e => setUsername(e.target.value)} required className="w-full px-2 py-1 bg-slate-950/60 border border-slate-700 rounded text-slate-50" />
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Nome</label>
              <input value={displayName} onChange={e => setDisplayName(e.target.value)} className="w-full px-2 py-1 bg-slate-950/60 border border-slate-700 rounded text-slate-50" />
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Senha</label>
              <input value={password} onChange={e => setPassword(e.target.value)} required type="password" className="w-full px-2 py-1 bg-slate-950/60 border border-slate-700 rounded text-slate-50" />
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Papel</label>
              <select value={role} onChange={e => setRole(e.target.value)} className="w-full px-2 py-1 bg-slate-950/60 border border-slate-700 rounded text-slate-50">
                <option value="viewer">viewer</option>
                <option value="admin">admin</option>
              </select>
            </div>
            <button type="submit" className="w-full py-2 bg-sky-600 rounded text-white">Criar</button>
          </form>
        </section>
      </main>
    </div>
  );
};

export default UsersPage;
