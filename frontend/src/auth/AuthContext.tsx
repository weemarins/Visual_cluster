import React, { createContext, useContext, useEffect, useState } from 'react';
import { apiClient } from '../services/api';

type AuthContextType = {
  token: string | null;
  role: string | null;
  username: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  initialized: boolean;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const STORAGE_KEY = 'vkube_auth';

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(null);
  const [role, setRole] = useState<string | null>(null);
  const [username, setUsername] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        const parsed = JSON.parse(stored) as { token: string; role: string; username: string };
        setToken(parsed.token);
        setRole(parsed.role);
        setUsername(parsed.username);
      } catch {
        localStorage.removeItem(STORAGE_KEY);
      }
    }
    setInitialized(true);
  }, []);

  const login = async (username: string, password: string) => {
    const res = await apiClient.post('/auth/login', { username, password });
    const data = res.data as { token: string; role: string; username: string };
    setToken(data.token);
    setRole(data.role);
    setUsername(data.username);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  };

  const logout = () => {
    setToken(null);
    setRole(null);
    setUsername(null);
    localStorage.removeItem(STORAGE_KEY);
    // Navegar para /login substituindo o histórico e forçar reload
    // Isso evita que o botão "voltar" traga de volta uma página com estado previamente em memória
    window.location.replace('/login');
  };

  useEffect(() => {
    // Lidando com bfcache / pageshow (voltar do navegador mantém estado)
    const onPageShow = (evt: PageTransitionEvent) => {
      // quando a página é exibida (incluindo restore do bfcache), re-sincronizamos com localStorage
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        try {
          const parsed = JSON.parse(stored) as { token: string; role: string; username: string };
          setToken(parsed.token);
          setRole(parsed.role);
          setUsername(parsed.username);
        } catch {
          setToken(null);
          setRole(null);
          setUsername(null);
          localStorage.removeItem(STORAGE_KEY);
        }
      } else {
        setToken(null);
        setRole(null);
        setUsername(null);
      }
      setInitialized(true);
    };

    window.addEventListener('pageshow', onPageShow);
    return () => window.removeEventListener('pageshow', onPageShow);
  }, []);

  useEffect(() => {
    if (token) {
      apiClient.defaults.headers.common.Authorization = `Bearer ${token}`;
    } else {
      delete apiClient.defaults.headers.common.Authorization;
    }
  }, [token]);

  return (
    <AuthContext.Provider value={{ token, role, username, login, logout, initialized }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth deve ser usado dentro de AuthProvider');
  }
  return ctx;
};

