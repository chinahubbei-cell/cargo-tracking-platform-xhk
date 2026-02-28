import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Organizations from './pages/Organizations';
import Orders from './pages/Orders';
import Devices from './pages/Devices';
import 'antd/dist/reset.css';

// 路由守卫
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const { isAuthenticated } = useAuth();
    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }
    return <>{children}</>;
};

const AppRoutes: React.FC = () => {
    return (
        <Routes>
            <Route path="/login" element={<Login />} />
            <Route
                path="/"
                element={
                    <ProtectedRoute>
                        <MainLayout />
                    </ProtectedRoute>
                }
            >
                <Route index element={<Dashboard />} />
                <Route path="orgs" element={<Organizations />} />
                <Route path="orders" element={<Orders />} />
                <Route path="devices" element={<Devices />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
    );
};

const App: React.FC = () => {
    return (
        <ConfigProvider locale={zhCN}>
            <BrowserRouter>
                <AuthProvider>
                    <AppRoutes />
                </AuthProvider>
            </BrowserRouter>
        </ConfigProvider>
    );
};

export default App;
