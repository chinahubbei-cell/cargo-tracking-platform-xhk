import React from 'react';
import { useNavigate } from 'react-router-dom';
import {
    PlusOutlined,
    SearchOutlined,
    SyncOutlined,
    SettingOutlined,
    FileTextOutlined,
    SendOutlined,
} from '@ant-design/icons';

interface QuickActionItem {
    key: string;
    label: string;
    icon: React.ReactNode;
    description: string;
    path?: string;
    onClick?: () => void;
    color: string;
}

interface QuickActionsProps {
    onRefresh?: () => void;
}

const QuickActions: React.FC<QuickActionsProps> = ({ onRefresh }) => {
    const navigate = useNavigate();

    const actions: QuickActionItem[] = [
        {
            key: 'new-shipment',
            label: '新建运单',
            icon: <PlusOutlined />,
            description: '创建新的货运订单',
            path: '/business/shipments?action=new',
            color: '#3b82f6',
        },
        {
            key: 'track-shipment',
            label: '追踪查询',
            icon: <SearchOutlined />,
            description: '查询货物实时位置',
            path: '/business/tracking',
            color: '#06b6d4',
        },
        {
            key: 'sync-devices',
            label: '同步设备',
            icon: <SyncOutlined />,
            description: '从云端同步设备数据',
            onClick: onRefresh,
            color: '#10b981',
        },
        {
            key: 'export-report',
            label: '导出报表',
            icon: <FileTextOutlined />,
            description: '生成运营统计报表',
            color: '#8b5cf6',
        },
        {
            key: 'send-alert',
            label: '发送通知',
            icon: <SendOutlined />,
            description: '群发运营通知',
            color: '#f59e0b',
        },
        {
            key: 'settings',
            label: '系统设置',
            icon: <SettingOutlined />,
            description: '配置系统参数',
            path: '/settings',
            color: '#64748b',
        },
    ];

    const handleAction = (action: QuickActionItem) => {
        if (action.onClick) {
            action.onClick();
        } else if (action.path) {
            navigate(action.path);
        }
    };

    return (
        <div className="glass-card">
            <h3 className="text-lg font-semibold text-slate-100 mb-4">快捷操作</h3>

            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                {actions.map((action, index) => (
                    <button
                        key={action.key}
                        onClick={() => handleAction(action)}
                        className={`
                            group relative p-4 rounded-xl
                            bg-slate-800/40 hover:bg-slate-800/70
                            border border-slate-700/50 hover:border-opacity-100
                            transition-all duration-300 text-left
                            animate-fade-in
                        `}
                        style={{
                            animationDelay: `${index * 50}ms`,
                            ['--hover-border-color' as any]: action.color,
                        }}
                    >
                        {/* 图标 */}
                        <div
                            className="w-10 h-10 rounded-lg flex items-center justify-center mb-3 
                                       transition-transform duration-300 group-hover:scale-110"
                            style={{
                                backgroundColor: `${action.color}20`,
                                color: action.color,
                            }}
                        >
                            <span className="text-xl">{action.icon}</span>
                        </div>

                        {/* 标签 */}
                        <p className="text-sm font-medium text-slate-200 m-0 mb-1">
                            {action.label}
                        </p>
                        <p className="text-xs text-slate-500 m-0 line-clamp-2">
                            {action.description}
                        </p>

                        {/* 悬停效果渐变 */}
                        <div
                            className="absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 
                                       transition-opacity duration-300 pointer-events-none"
                            style={{
                                background: `radial-gradient(circle at 50% 0%, ${action.color}10, transparent 70%)`,
                            }}
                        />
                    </button>
                ))}
            </div>
        </div>
    );
};

export default QuickActions;
