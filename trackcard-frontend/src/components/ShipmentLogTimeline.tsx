import React, { useEffect, useState } from 'react';
import { Timeline, Spin, Empty, Tag } from 'antd';
import {
    PlusCircleOutlined,
    EditOutlined,
    SwapOutlined,
    CheckCircleOutlined,
    DeleteOutlined,
    LinkOutlined,
    DisconnectOutlined,
    EnvironmentOutlined,
    RocketOutlined,
    CarOutlined,
} from '@ant-design/icons';
import api from '../api/client';
import type { ShipmentLog } from '../types';

interface ShipmentLogTimelineProps {
    shipmentId: string;
    visible?: boolean;
}

const actionIcons: Record<string, React.ReactNode> = {
    created: <PlusCircleOutlined style={{ color: '#52c41a' }} />,
    updated: <EditOutlined style={{ color: '#1890ff' }} />,
    status_changed: <SwapOutlined style={{ color: '#faad14' }} />,
    device_bound: <LinkOutlined style={{ color: '#52c41a' }} />,
    device_unbound: <DisconnectOutlined style={{ color: '#ff4d4f' }} />,
    device_replaced: <SwapOutlined style={{ color: '#722ed1' }} />,
    deleted: <DeleteOutlined style={{ color: '#ff4d4f' }} />,
    geofence_trigger: <EnvironmentOutlined style={{ color: '#13c2c2' }} />,
    stage_transition: <RocketOutlined style={{ color: '#eb2f96' }} />,
    delivered: <CarOutlined style={{ color: '#52c41a' }} />,
};

const actionColors: Record<string, string> = {
    created: 'green',
    updated: 'blue',
    status_changed: 'orange',
    device_bound: 'green',
    device_unbound: 'red',
    device_replaced: 'purple',
    deleted: 'red',
    geofence_trigger: 'cyan',
    stage_transition: 'magenta',
    delivered: 'green',
};

const ShipmentLogTimeline: React.FC<ShipmentLogTimelineProps> = ({ shipmentId, visible = true }) => {
    const [loading, setLoading] = useState(false);
    const [logs, setLogs] = useState<ShipmentLog[]>([]);

    useEffect(() => {
        if (visible && shipmentId) {
            loadLogs();
        }
    }, [shipmentId, visible]);

    const loadLogs = async () => {
        setLoading(true);
        try {
            const res = await api.getShipmentLogs(shipmentId);
            setLogs(res.data || []);
        } catch {
            setLogs([]);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div style={{ textAlign: 'center', padding: 40 }}>
                <Spin size="large" />
            </div>
        );
    }

    if (logs.length === 0) {
        return <Empty description="暂无操作日志" />;
    }

    return (
        <Timeline
            items={logs.map((log) => ({
                dot: actionIcons[log.action] || <CheckCircleOutlined />,
                children: (
                    <div>
                        <div style={{ marginBottom: 4 }}>
                            <Tag color={actionColors[log.action] || 'default'} style={{ marginRight: 8 }}>
                                {log.action === 'created' ? '创建' :
                                    log.action === 'updated' ? '更新' :
                                        log.action === 'status_changed' ? '状态变更' :
                                            log.action === 'device_bound' ? '绑定设备' :
                                                log.action === 'device_unbound' ? '解绑设备' :
                                                    log.action === 'device_replaced' ? '更换设备' :
                                                        log.action === 'geofence_trigger' ? '围栏触发' :
                                                            log.action === 'stage_transition' ? '环节变更' :
                                                                log.action === 'delivered' ? '签收' :
                                                                    log.action === 'deleted' ? '删除' : log.action}
                            </Tag>
                            <span style={{ fontSize: 12, color: '#999' }}>
                                {new Date(log.created_at).toLocaleString('zh-CN')}
                            </span>
                        </div>
                        <div style={{ color: '#333' }}>{log.description}</div>
                        {log.operator_id && (
                            <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                                操作人: {log.operator_id} | IP: {log.operator_ip || '-'}
                            </div>
                        )}
                    </div>
                ),
            }))}
        />
    );
};

export default ShipmentLogTimeline;
