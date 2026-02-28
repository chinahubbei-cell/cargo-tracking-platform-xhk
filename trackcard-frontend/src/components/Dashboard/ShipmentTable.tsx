import React from 'react';
import type { Shipment } from '../../types';
import { useNavigate } from 'react-router-dom';

interface ShipmentTableProps {
    shipments: Shipment[];
    maxRows?: number;
    onViewAll?: () => void;
}

const getStatusConfig = (status: string) => {
    switch (status) {
        case 'delivered':
            return { label: '已送达', className: 'delivered' };
        case 'in_transit':
            return { label: '运输中', className: 'in-transit' };
        case 'pending':
            return { label: '待发运', className: 'pending' };
        case 'cancelled':
            return { label: '已取消', className: 'delayed' };
        default:
            return { label: status, className: 'pending' };
    }
};

// 模拟数据（使用新运单号格式）
const mockShipments: Shipment[] = [
    {
        id: '260116000001',
        origin: '深圳',
        destination: '伦敦',
        status: 'delivered',
        progress: 100,
        eta: '2024-01-15T10:30:00Z',
        created_at: '2024-01-10T08:00:00Z',
    },
    {
        id: '260116000002',
        origin: '上海',
        destination: '纽约',
        status: 'in_transit',
        progress: 65,
        eta: '2024-01-18T14:00:00Z',
        created_at: '2024-01-12T10:00:00Z',
    },
    {
        id: '260116000003',
        origin: '广州',
        destination: '巴黎',
        status: 'in_transit',
        progress: 40,
        eta: '2024-01-20T09:00:00Z',
        created_at: '2024-01-14T06:00:00Z',
    },
    {
        id: '260116000004',
        origin: '北京',
        destination: '东京',
        status: 'pending',
        progress: 0,
        eta: '2024-01-22T12:00:00Z',
        created_at: '2024-01-14T14:00:00Z',
    },
];

const ShipmentTable: React.FC<ShipmentTableProps> = ({
    shipments,
    maxRows = 4,
    onViewAll,
}) => {
    const navigate = useNavigate();
    const displayShipments = (shipments.length > 0 ? shipments : mockShipments).slice(0, maxRows);

    const formatDate = (dateStr?: string) => {
        if (!dateStr) return '-';
        const date = new Date(dateStr);
        return date.toLocaleDateString('zh-CN', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    return (
        <div className="card" style={{ height: '100%' }}>
            <div className="card-header">
                <div>
                    <h3 className="card-title">最近运单</h3>
                    <p className="card-subtitle">Recent Shipments</p>
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

            <div style={{ overflow: 'auto' }}>
                <table className="data-table">
                    <thead>
                        <tr>
                            <th>运单号</th>
                            <th>始发地</th>
                            <th>目的地</th>
                            <th>状态</th>
                            <th>预计送达</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {displayShipments.map((shipment, index) => {
                            const statusConfig = getStatusConfig(shipment.status);
                            return (
                                <tr
                                    key={shipment.id}
                                    className="animate-fade-in"
                                    style={{ animationDelay: `${index * 50}ms` }}
                                >
                                    <td>
                                        <span style={{ fontWeight: 500, color: 'var(--text-primary)' }}>
                                            {shipment.id}
                                        </span>
                                    </td>
                                    <td>{shipment.origin}</td>
                                    <td>{shipment.destination}</td>
                                    <td>
                                        <span className={`status-tag ${statusConfig.className}`}>
                                            <span className="dot" />
                                            {statusConfig.label}
                                        </span>
                                    </td>
                                    <td>{formatDate(shipment.eta)}</td>
                                    <td>
                                        <button
                                            onClick={() => navigate(`/shipments?id=${shipment.id}`)}
                                            className="btn btn-ghost"
                                            style={{ padding: '4px 8px', fontSize: 12 }}
                                        >
                                            详情
                                        </button>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default ShipmentTable;
