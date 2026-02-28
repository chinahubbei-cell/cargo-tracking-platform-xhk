import React from 'react';
import {
    RocketOutlined,
    CheckCircleOutlined,
    WarningOutlined,
    BellOutlined,
    EnvironmentOutlined,
    ThunderboltOutlined,
} from '@ant-design/icons';
import type { Alert } from '../../types';

interface ActivityItem {
    id: string;
    type: 'shipment' | 'alert' | 'device' | 'system';
    severity?: 'info' | 'warning' | 'critical' | 'success';
    title: string;
    description?: string;
    time: string;
    location?: string;
}

interface ActivityTimelineProps {
    alerts?: Alert[];
    maxItems?: number;
    onViewAll?: () => void;
}

const getActivityIcon = (type: string, severity?: string) => {
    if (type === 'alert') {
        if (severity === 'critical') return <WarningOutlined />;
        if (severity === 'warning') return <BellOutlined />;
        return <BellOutlined />;
    }
    if (type === 'shipment') return <RocketOutlined />;
    if (type === 'device') return <ThunderboltOutlined />;
    return <CheckCircleOutlined />;
};

const getActivityColor = (type: string, severity?: string) => {
    if (type === 'alert') {
        if (severity === 'critical') return { bg: 'bg-rose-500/20', text: 'text-rose-400', border: 'border-rose-500/50' };
        if (severity === 'warning') return { bg: 'bg-amber-500/20', text: 'text-amber-400', border: 'border-amber-500/50' };
        return { bg: 'bg-blue-500/20', text: 'text-blue-400', border: 'border-blue-500/50' };
    }
    if (type === 'shipment') return { bg: 'bg-cyan-500/20', text: 'text-cyan-400', border: 'border-cyan-500/50' };
    if (type === 'device') return { bg: 'bg-purple-500/20', text: 'text-purple-400', border: 'border-purple-500/50' };
    return { bg: 'bg-emerald-500/20', text: 'text-emerald-400', border: 'border-emerald-500/50' };
};

const formatTimeAgo = (time: string) => {
    const now = new Date();
    const then = new Date(time);
    const diffMs = now.getTime() - then.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMins < 1) return '刚刚';
    if (diffMins < 60) return `${diffMins} 分钟前`;
    if (diffHours < 24) return `${diffHours} 小时前`;
    if (diffDays < 7) return `${diffDays} 天前`;
    return then.toLocaleDateString('zh-CN');
};

// 模拟活动数据
const generateMockActivities = (): ActivityItem[] => [
    {
        id: '1',
        type: 'shipment',
        severity: 'success',
        title: '运单 #XHK-8823 已从深圳发出',
        description: '目的地：洛杉矶港',
        time: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
        location: '深圳',
    },
    {
        id: '2',
        type: 'alert',
        severity: 'critical',
        title: '设备 #XHK-002 电量告警',
        description: '当前电量: 15%',
        time: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
    },
    {
        id: '3',
        type: 'shipment',
        severity: 'success',
        title: '运单 #XHK-9921 已抵达洛杉矶港',
        time: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
        location: '洛杉矶',
    },
    {
        id: '4',
        type: 'device',
        severity: 'info',
        title: '设备 #XHK-015 上线',
        description: '信号强度: 优',
        time: new Date(Date.now() - 8 * 60 * 60 * 1000).toISOString(),
    },
    {
        id: '5',
        type: 'alert',
        severity: 'warning',
        title: '运单 #XHK-7762 可能延误',
        description: '预计延误: 2小时',
        time: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(),
    },
];

const ActivityTimeline: React.FC<ActivityTimelineProps> = ({
    alerts,
    maxItems = 5,
    onViewAll,
}) => {
    // 将告警转换为活动项，或使用模拟数据
    const activities: ActivityItem[] = alerts && alerts.length > 0
        ? alerts.slice(0, maxItems).map(alert => ({
            id: alert.id,
            type: 'alert' as const,
            severity: alert.severity,
            title: alert.title,
            description: alert.message,
            time: alert.created_at,
            location: alert.location,
        }))
        : generateMockActivities().slice(0, maxItems);

    return (
        <div className="glass-card h-full flex flex-col">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-slate-100 m-0">实时动态</h3>
                {onViewAll && (
                    <button
                        onClick={onViewAll}
                        className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                    >
                        查看全部 →
                    </button>
                )}
            </div>

            <div className="flex-1 overflow-y-auto custom-scrollbar space-y-1">
                {activities.map((activity, index) => {
                    const colors = getActivityColor(activity.type, activity.severity);
                    const isLast = index === activities.length - 1;

                    return (
                        <div
                            key={activity.id}
                            className={`
                                relative pl-8 pb-4
                                ${!isLast ? 'border-l-2 border-slate-700/50 ml-3' : 'ml-3'}
                                animate-fade-in
                            `}
                            style={{ animationDelay: `${index * 100}ms` }}
                        >
                            {/* 时间线节点 */}
                            <div className={`
                                absolute left-[-9px] top-0 w-5 h-5 rounded-full
                                flex items-center justify-center text-xs
                                ${colors.bg} ${colors.text} ${colors.border} border
                            `}>
                                {getActivityIcon(activity.type, activity.severity)}
                            </div>

                            {/* 内容卡片 */}
                            <div className={`
                                p-3 rounded-lg bg-slate-800/40 hover:bg-slate-800/60
                                border border-transparent hover:${colors.border}
                                transition-all duration-200 cursor-pointer
                            `}>
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex-1 min-w-0">
                                        <p className="text-sm font-medium text-slate-200 m-0 truncate">
                                            {activity.title}
                                        </p>
                                        {activity.description && (
                                            <p className="text-xs text-slate-400 m-0 mt-1 truncate">
                                                {activity.description}
                                            </p>
                                        )}
                                    </div>
                                    <span className="text-xs text-slate-500 whitespace-nowrap flex-shrink-0">
                                        {formatTimeAgo(activity.time)}
                                    </span>
                                </div>
                                {activity.location && (
                                    <div className="flex items-center gap-1 mt-2 text-xs text-slate-500">
                                        <EnvironmentOutlined style={{ fontSize: '10px' }} />
                                        <span>{activity.location}</span>
                                    </div>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>

            {activities.length === 0 && (
                <div className="flex-1 flex items-center justify-center text-slate-500">
                    <p>暂无动态</p>
                </div>
            )}
        </div>
    );
};

export default ActivityTimeline;
