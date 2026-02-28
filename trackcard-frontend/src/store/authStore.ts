import { create } from 'zustand';
import type { User, UserOrg } from '../types';

interface AuthState {
    token: string | null;
    user: User | null;
    isAuthenticated: boolean;
    currentOrgId: string | null;  // 当前选中的组织ID
    currentOrg: UserOrg | null;   // 当前选中的组织信息
    setAuth: (token: string, user: User) => void;
    logout: () => void;
    init: () => void;
    setCurrentOrg: (orgId: string | null) => void;
    getCurrentOrgId: () => string | null;
}

export const useAuthStore = create<AuthState>((set, get) => ({
    token: null,
    user: null,
    isAuthenticated: false,
    currentOrgId: null,
    currentOrg: null,
    setAuth: (token: string, user: User) => {
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify(user));
        // 默认选择主部门组织
        let defaultOrgId: string | null = null;
        let defaultOrg: UserOrg | null = null;
        if (user.organizations && user.organizations.length > 0) {
            const primaryOrg = user.organizations.find(o => o.is_primary);
            defaultOrg = primaryOrg || user.organizations[0];
            defaultOrgId = defaultOrg.id;
            localStorage.setItem('currentOrgId', defaultOrgId);
        }
        set({ token, user, isAuthenticated: true, currentOrgId: defaultOrgId, currentOrg: defaultOrg });
    },
    logout: () => {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('currentOrgId');
        set({ token: null, user: null, isAuthenticated: false, currentOrgId: null, currentOrg: null });
    },
    init: () => {
        const token = localStorage.getItem('token');
        const userStr = localStorage.getItem('user');
        const savedOrgId = localStorage.getItem('currentOrgId');
        if (token && userStr) {
            try {
                const user = JSON.parse(userStr) as User;
                let currentOrgId: string | null = savedOrgId;
                let currentOrg: UserOrg | null = null;

                // 验证保存的orgId是否仍属于该用户
                if (user.organizations && user.organizations.length > 0) {
                    if (savedOrgId) {
                        currentOrg = user.organizations.find(o => o.id === savedOrgId) || null;
                    }
                    if (!currentOrg) {
                        // 如果没有保存的组织或无效，使用主部门
                        const primaryOrg = user.organizations.find(o => o.is_primary);
                        currentOrg = primaryOrg || user.organizations[0];
                        currentOrgId = currentOrg.id;
                        localStorage.setItem('currentOrgId', currentOrgId);
                    }
                }
                set({ token, user, isAuthenticated: true, currentOrgId, currentOrg });
            } catch {
                localStorage.removeItem('token');
                localStorage.removeItem('user');
                localStorage.removeItem('currentOrgId');
            }
        }
    },
    setCurrentOrg: (orgId: string | null) => {
        const { user } = get();
        let currentOrg: UserOrg | null = null;
        if (orgId && user?.organizations) {
            currentOrg = user.organizations.find(o => o.id === orgId) || null;
        }
        if (orgId) {
            localStorage.setItem('currentOrgId', orgId);
        } else {
            localStorage.removeItem('currentOrgId');
        }
        set({ currentOrgId: orgId, currentOrg });
    },
    getCurrentOrgId: () => {
        return get().currentOrgId;
    },
}));

// Initialize on load
if (typeof window !== 'undefined') {
    useAuthStore.getState().init();
}
