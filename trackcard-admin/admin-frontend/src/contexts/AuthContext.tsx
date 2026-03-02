import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import axios from 'axios';

interface UserOrg {
    id: string;
    name: string;
    is_primary?: boolean;
    position?: string;
}

interface User {
    id: string;
    username?: string;
    name: string;
    email: string;
    role: string;
    organizations?: UserOrg[];
}

interface AuthContextType {
    user: User | null;
    token: string | null;
    login: (username: string, password: string) => Promise<boolean>;
    loginBySMS: (phone: string, code: string) => Promise<{ success: boolean; needSelectOrg?: boolean; orgs?: UserOrg[] }>;
    selectOrg: (orgID: string) => Promise<boolean>;
    sendSMSCode: (phone: string, scene?: 'login' | 'reset_password') => Promise<{ ok: boolean; debugCode?: string }>;
    logout: () => void;
    isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) throw new Error('useAuth must be used within an AuthProvider');
    return context;
};

interface AuthProviderProps {
    children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
    const [user, setUser] = useState<User | null>(null);
    const [token, setToken] = useState<string | null>(localStorage.getItem('admin_token'));

    useEffect(() => {
        if (token) {
            axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
            fetchUser();
        }
    }, [token]);

    const fetchUser = async () => {
        try {
            const res = await axios.get('/api/auth/me');
            if (res.data.success) setUser(res.data.data);
        } catch {
            logout();
        }
    };

    const login = async (username: string, password: string): Promise<boolean> => {
        try {
            const res = await axios.post('/api/auth/login', { email: username, password });
            if (res.data.success) {
                const data = res.data.data || res.data;
                if (data.token) {
                    localStorage.setItem('admin_token', data.token);
                    axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`;
                    setToken(data.token);
                    if (data.user) setUser(data.user); else await fetchUser();
                    return true;
                }
            }
        } catch {
            return false;
        }
        return false;
    };

    const sendSMSCode = async (phone: string, scene: 'login' | 'reset_password' = 'login'): Promise<{ ok: boolean; debugCode?: string }> => {
        try {
            const res = await axios.post('/api/auth/sms/send-code', { phone_number: phone, scene });
            const data = res.data?.data || res.data;
            return { ok: !!res.data.success, debugCode: data?.debug_code };
        } catch {
            return { ok: false };
        }
    };

    const loginBySMS = async (phone: string, code: string): Promise<{ success: boolean; needSelectOrg?: boolean; orgs?: UserOrg[] }> => {
        try {
            const res = await axios.post('/api/auth/sms/login', { phone_number: phone, code });
            if (res.data.success) {
                const data = res.data.data || res.data;
                if (data.need_select_org) {
                    if (data.token_temp) {
                        localStorage.setItem('admin_token', data.token_temp);
                        axios.defaults.headers.common['Authorization'] = `Bearer ${data.token_temp}`;
                        setToken(data.token_temp);
                    }
                    return { success: true, needSelectOrg: true, orgs: data.orgs || [] };
                }
                if (data.token) {
                    localStorage.setItem('admin_token', data.token);
                    axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`;
                    setToken(data.token);
                    if (data.user) setUser(data.user); else await fetchUser();
                    return { success: true };
                }
            }
        } catch {
            return { success: false };
        }
        return { success: false };
    };

    const selectOrg = async (orgID: string): Promise<boolean> => {
        try {
            const res = await axios.post('/api/auth/select-org', { org_id: orgID });
            if (res.data.success) {
                const data = res.data.data || res.data;
                if (data.token) {
                    localStorage.setItem('admin_token', data.token);
                    axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`;
                    setToken(data.token);
                    await fetchUser();
                    return true;
                }
            }
        } catch {
            return false;
        }
        return false;
    };

    const logout = () => {
        localStorage.removeItem('admin_token');
        delete axios.defaults.headers.common['Authorization'];
        setToken(null);
        setUser(null);
    };

    return (
        <AuthContext.Provider value={{ user, token, login, loginBySMS, selectOrg, sendSMSCode, logout, isAuthenticated: !!user }}>
            {children}
        </AuthContext.Provider>
    );
};
