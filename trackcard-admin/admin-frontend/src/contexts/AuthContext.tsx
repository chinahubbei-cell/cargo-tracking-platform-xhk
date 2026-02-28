import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import axios from 'axios';

interface User {
    id: string;
    username: string;
    name: string;
    email: string;
    role: string;
}

interface AuthContextType {
    user: User | null;
    token: string | null;
    login: (username: string, password: string) => Promise<boolean>;
    logout: () => void;
    isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
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
            if (res.data.success) {
                setUser(res.data.data);
            }
        } catch {
            logout();
        }
    };

    const login = async (username: string, password: string): Promise<boolean> => {
        try {
            const res = await axios.post('/api/auth/login', { email: username, password });
            if (res.data.success) {
                // 后端返回格式: { success: true, data: { token, user } }
                // 或者直接返回 { token, user } (取决于具体实现，但根据utils.SuccessResponse是前者)
                const data = res.data.data || res.data;

                if (data.token) {
                    localStorage.setItem('admin_token', data.token);
                    axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`;
                    setToken(data.token);
                    // 设置用户信息
                    if (data.user) {
                        setUser(data.user);
                    } else {
                        // 如果没有返回用户信息，再请求一次
                        await fetchUser();
                    }
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
        <AuthContext.Provider value={{ user, token, login, logout, isAuthenticated: !!user }}>
            {children}
        </AuthContext.Provider>
    );
};
