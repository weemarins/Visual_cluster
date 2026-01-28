import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const LoginPage: React.FC = () => {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await login(username, password);
      navigate('/clusters');
    } catch (err) {
      setError('Falha no login. Verifique usuário e senha.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-slate-950 to-slate-900">
      <div className="w-full max-w-md bg-slate-900/80 border border-slate-700 rounded-2xl shadow-2xl p-8 space-y-6">
        <h1 className="text-2xl font-semibold text-slate-50 text-center">Visual Kubernetes Topology</h1>
        <p className="text-sm text-slate-400 text-center">Autenticação via LDAP</p>
        <form className="space-y-4" onSubmit={handleSubmit}>
          <div>
            <label className="block text-sm font-medium text-slate-200 mb-1">Usuário</label>
            <input
              type="text"
              className="w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 text-slate-50 focus:outline-none focus:ring-2 focus:ring-sky-500"
              value={username}
              onChange={e => setUsername(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200 mb-1">Senha</label>
            <input
              type="password"
              className="w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 text-slate-50 focus:outline-none focus:ring-2 focus:ring-sky-500"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
            />
          </div>
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-sky-600 hover:bg-sky-500 disabled:bg-slate-700 disabled:cursor-not-allowed py-2 font-semibold text-slate-50 transition-colors"
          >
            {loading ? 'Entrando...' : 'Entrar'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default LoginPage;

