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

// 组织
// 组织
export const orgApi = {
    list: (params?: any) => api.get('/orgs', { params }),
    create: (data: any) => api.post('/orgs', data),
    get: (id: string) => api.get(`/orgs/${id}`),
    update: (id: string, data: any) => api.put(`/orgs/${id}`, data),
    delete: (id: string) => api.delete(`/orgs/${id}`),
    setService: (id: string, data: any) => api.put(`/orgs/${id}/service`, data),
    renew: (id: string, data: any) => api.post(`/orgs/${id}/renew`, data),
    getExpiring: () => api.get('/orgs/expiring'),
};

// 订单（V2 全链路）
export const orderApi = {
    list: (params?: any) => api.get('/orders', { params }),
    create: (data: any) => api.post('/orders', data),
    get: (id: string) => api.get(`/orders/${id}`),
    submitReview: (id: string) => api.post(`/orders/${id}/submit-review`),
    review: (id: string, data: { action: 'approve' | 'reject'; comment: string }) =>
        api.post(`/orders/${id}/review`, data),
    generateContract: (id: string) => api.post(`/orders/${id}/contract/generate`),
    confirmOfflineSign: (id: string, data?: { file_url?: string }) =>
        api.post(`/orders/${id}/contract/confirm-offline`, data),
    confirmPayment: (id: string, data?: { amount?: number; note?: string }) =>
        api.post(`/orders/${id}/payment/confirm`, data),
    void: (id: string, data?: { reason?: string }) =>
        api.post(`/orders/${id}/void`, data),
    invoiceReview: (id: string, data: { action: 'approve' | 'reject'; comment?: string }) =>
        api.post(`/orders/${id}/invoice/review`, data),
    issueInvoice: (id: string, data: { invoice_no: string; file_url?: string }) =>
        api.post(`/orders/${id}/invoice/issue`, data),
    startFulfilling: (id: string) => api.post(`/orders/${id}/fulfilling`),
    complete: (id: string) => api.post(`/orders/${id}/complete`),
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
