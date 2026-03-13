import axios from 'axios';

const api = axios.create({
    baseURL: '/api/admin',
    timeout: 10000,
});

// 请求拦截器
api.interceptors.request.use((config) => {
    const token = localStorage.getItem('admin_token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// 响应拦截器
api.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('admin_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

// 认证
export const authApi = {
    login: (username: string, password: string) =>
        api.post('/auth/login', { username, password }),
    getMe: () => api.get('/me'),
};

// 客户
export const orgApi = {
    list: (params?: any) => api.get('/orgs', { params }),
    create: (data: any) => api.post('/orgs', data),
    get: (id: string) => api.get(`/orgs/${id}`),
    update: (id: string, data: any) => api.put(`/orgs/${id}`, data),
    delete: (id: string) => api.delete(`/orgs/${id}`),
    setService: (id: string, data: any) => api.put(`/orgs/${id}/service`, data),
    renew: (id: string, data: any) => api.post(`/orgs/${id}/renew`, data),
    getExpiring: () => api.get('/orgs/expiring'),
    getRenewals: (id: string) => api.get(`/orgs/${id}/renewals`),
    getDevices: (id: string) => api.get(`/orgs/${id}/devices`),
    getStats: (id: string) => api.get(`/orgs/${id}/stats`),
};

// 设备
export const deviceApi = {
    list: (params?: any) => api.get('/devices', { params }),
    create: (data: any) => api.post('/devices', data),
    batchImport: (data: any) => api.post('/devices/batch-import', data),
    get: (id: string) => api.get(`/devices/${id}`),
    update: (id: string, data: any) => api.put(`/devices/${id}`, data),
    allocate: (id: string, data: any) => api.put(`/devices/${id}/allocate`, data),
    return: (id: string, data: any) => api.put(`/devices/${id}/return`, data),
    stats: () => api.get('/devices/stats'),
    logs: (params?: any) => api.get('/devices/logs', { params }),
};

// 仪表盘
export const dashboardApi = {
    getStats: () => api.get('/dashboard'),
};

export default api;
