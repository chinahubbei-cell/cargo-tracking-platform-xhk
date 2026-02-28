import React from 'react';
import type { Alert } from '../../types';
import {
    ClockCircleOutlined,
    ThunderboltOutlined,
    NodeIndexOutlined,
    FileProtectOutlined,
    AppstoreOutlined
} from '@ant-design/icons';

interface AlertListProps {
    alerts: Alert[];
    maxItems?: number;
    onViewAll?: () => void;
    onAlertClick?: (alert: Alert) => void;
}

const formatTimeAgo = (time: string) => {
    const now = new Date();
    const then = new Date(time);
    const diffMs = now.getTime() - then.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

    if (diffMins < 1) return '刚刚';
    if (diffMins < 60) return `${diffMins}分钟前`;
    if (diffHours < 24) return `${diffHours}小时前`;
    return then.toLocaleDateString('zh-CN');
};

const getSeverityLabel = (severity: string) => {
    switch (severity) {
        case 'critical': return '紧急';
        case 'warning': return '警告';
        default: return '消息';
    }
};

const getCategoryIcon = (category: string = 'system') => {
    switch (category) {
        case 'physical': return <ThunderboltOutlined style={{ color: '#fa8c16' }} />; // 物理环境
        case 'node': return <NodeIndexOutlined style={{ color: '#1890ff' }} />;      // 节点异动
        case 'operation': return <FileProtectOutlined style={{ color: '#722ed1' }} />; // 末端作业
        default: return <AppstoreOutlined style={{ color: '#8c8c8c' }} />;
    }
};

const getCategoryLabel = (category: string = 'system') => {
    switch (category) {
        case 'physical': return '物理环境';
        case 'node': return '节点异动';
        case 'operation': return '关务作业';
        default: return '系统';
    }
};

const AlertList: React.FC<AlertListProps> = ({
    alerts,
    maxItems = 5,
    onViewAll,
    onAlertClick,
}) => {
    const displayAlerts = alerts.slice(0, maxItems);

    return (
        <div className="card" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <div className="card-header">
                <div>
                    <h3 className="card-title">系统告警</h3>
                    <p className="card-subtitle">System Alerts</p>
                </div>
                {onViewAll && (
                    <button
                        onClick={onViewAll}
                        className="btn btn-ghost"
                        style={{ padding: '6px 12px', fontSize: 13 }}
                    >
                        查看全部 →
                    </button>
                )}
            </div>

            <div style={{ flex: 1, overflow: 'auto' }}>
                {displayAlerts.length > 0 ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {displayAlerts.map((alert, index) => (
                            <div
                                key={alert.id}
                                className="alert-item animate-fade-in"
                                style={{ animationDelay: `${index * 50}ms` }}
                                onClick={() => onAlertClick?.(alert)}
                            >
                                <div style={{ display: 'flex', alignItems: 'center', marginRight: 12 }}>
                                    {getCategoryIcon(alert.category)}
                                </div>
                                <div className="alert-content">
                                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                                        <span className={`alert-badge ${alert.severity}`}>
                                            {getSeverityLabel(alert.severity)}
                                        </span>
                                        <span style={{ fontSize: 11, color: '#999' }}>
                                            {getCategoryLabel(alert.category)}
                                        </span>
                                    </div>
                                    <p className="alert-title">{alert.title}</p>
                                    <div className="alert-time">
                                        <ClockCircleOutlined style={{ marginRight: 4, fontSize: 11 }} />
                                        {formatTimeAgo(alert.created_at)}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <div style={{
                        height: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: 'var(--text-muted)',
                        fontSize: 14
                    }}>
                        暂无告警信息
                    </div>
                )}
            </div>

            {/* 状态分布指示器 */}
            {displayAlerts.length > 0 && (
                <div style={{
                    marginTop: 16,
                    paddingTop: 16,
                    borderTop: '1px solid var(--border-light)',
                    display: 'flex',
                    gap: 16,
                    fontSize: 12
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: 'var(--danger)'
                        }} />
                        <span style={{ color: 'var(--text-tertiary)' }}>
                            紧急 {alerts.filter(a => a.severity === 'critical').length}
                        </span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: 'var(--warning)'
                        }} />
                        <span style={{ color: 'var(--text-tertiary)' }}>
                            警告 {alerts.filter(a => a.severity === 'warning').length}
                        </span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: 'var(--info)'
                        }} />
                        <span style={{ color: 'var(--text-tertiary)' }}>
                            消息 {alerts.filter(a => a.severity === 'info').length}
                        </span>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AlertList;
