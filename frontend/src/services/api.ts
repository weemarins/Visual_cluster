import axios from 'axios';

const STORAGE_KEY = 'vkube_auth';

export const apiClient = axios.create({
  baseURL: '/api/v1'
});

// Garantir que o token utilizado seja sempre o presente no localStorage
apiClient.interceptors.request.use((config) => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as { token: string };
      if (parsed?.token) {
        config.headers = config.headers || {};
        (config.headers as any).Authorization = `Bearer ${parsed.token}`;
      }
    }
  } catch {
    // se parsing falhar, apenas continue sem header
  }
  return config;
});

