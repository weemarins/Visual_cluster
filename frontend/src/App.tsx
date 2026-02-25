import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import LoginPage from './pages/LoginPage';
import ClustersPage from './pages/ClustersPage';
import TopologyPage from './pages/TopologyPage';
import UsersPage from './pages/UsersPage';
import { AuthProvider, useAuth } from './auth/AuthContext';

const PrivateRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { token, initialized } = useAuth();
  if (!initialized) return null;
  if (!token) return <Navigate to="/login" replace />;
  return children;
};

const AdminRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { token, role, initialized } = useAuth();
  if (!initialized) return null;
  if (!token || role !== 'admin') return <Navigate to="/clusters" replace />;
  return children;
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/clusters"
          element={
            <PrivateRoute>
              <ClustersPage />
            </PrivateRoute>
          }
        />
        <Route
          path="/admin/users"
          element={
            <AdminRoute>
              <UsersPage />
            </AdminRoute>
          }
        />
        <Route
          path="/topology/:clusterId"
          element={
            <PrivateRoute>
              <TopologyPage />
            </PrivateRoute>
          }
        />
        <Route path="*" element={<Navigate to="/clusters" replace />} />
      </Routes>
    </AuthProvider>
  );
};

export default App;

