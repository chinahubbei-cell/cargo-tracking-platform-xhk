import React, { useState, useEffect, useCallback } from 'react';
import { Tooltip, Spin, Tag } from 'antd';
import {
    CheckCircleFilled,
    ClockCircleFilled,
    MinusCircleFilled,
    AimOutlined,
    ApiOutlined,
    UserOutlined,
    StopOutlined,
    EnvironmentOutlined,
} from '@ant-design/icons';
import api from '../api/client';
import type { ShipmentStage, StageStatus } from '../types';
import './TransportNodeTimeline.css';

// 停留记录接口
interface DeviceStopRecord {
    id: string;
    device_id: string;
    device_external_id: string;
    shipment_id: string;
    start_time: string;
    end_time: string | null;
    duration_seconds: number;
    duration_text: string;
    latitude: number | null;
    longitude: number | null;
    address: string;
    status: 'active' | 'completed';
    alert_sent: boolean;
    created_at: string;
}

interface TransportNodeTimelineProps {
    shipmentId: string;
    compact?: boolean;
    refreshInterval?: number; // 毫秒，默认5000
    onStageChange?: () => void;
}

// 节点图标映射
const StageIcons: Record<string, string> = {
    // 主运输环节7个
    pre_transit: '🚚',
    origin_port: '🚢',
    main_line: '🌊',
    transit_port: '⚓',
    dest_port: '🏁',
    last_mile: '🚚',
    delivered: '✅',
    // 硬件触发子事件
    origin_arrival: '🚢⬇️',
    origin_departure: '🚢⬆️',
    transit_arrival: '⚓⬇️',
    transit_departure: '⚓⬆️',
    dest_arrival: '🏁⬇️',
    dest_departure: '🏁⬆️',
    // 兼容旧版
    pickup: '🚚',
    first_mile: '📦',
    main_carriage: '🌊',
    delivery: '🚚',
};

// 默认节点配置（新7主环节）
const DEFAULT_STAGES = [
    { code: 'pre_transit', name: '前程运输' },
    { code: 'origin_port', name: '起运港' },
    { code: 'main_line', name: '干线运输' },
    { code: 'transit_port', name: '中转港' },
    { code: 'dest_port', name: '目的港' },
    { code: 'last_mile', name: '末端配送' },
    { code: 'delivered', name: '签收' },
];

// 触发类型图标
const TriggerIcons: Record<string, React.ReactNode> = {
    manual: <UserOutlined title="手动操作" />,
    geofence: <AimOutlined title="围栏触发" />,
    api: <ApiOutlined title="API回传" />,
};

// 触发类型标签
const TriggerLabels: Record<string, string> = {
    manual: '手动',
    geofence: '围栏',
    api: 'API',
};

// 状态颜色
const StatusColors: Record<StageStatus, string> = {
    pending: '#d9d9d9',
    in_progress: '#1890ff',
    completed: '#52c41a',
    skipped: '#faad14',
};

const TransportNodeTimeline: React.FC<TransportNodeTimelineProps> = ({
    shipmentId,
    compact = false,
    refreshInterval = 5000,
}) => {
    const [loading, setLoading] = useState(true);
    const [stages, setStages] = useState<ShipmentStage[]>([]);
    const [stopRecords, setStopRecords] = useState<DeviceStopRecord[]>([]);
    const [nowTs, setNowTs] = useState(() => Date.now());

    // 加载环节数据
    const loadStages = useCallback(async () => {
        if (!shipmentId) return;
        try {
            const response = await api.getShipmentStages(shipmentId);
            const data = response as any;
            // API returns { success: true, data: { stages: [...], current_stage: string } }
            const stagesData = data?.data?.stages || data?.stages;
            if (stagesData && stagesData.length > 0) {
                setStages(stagesData);
            }
        } catch (error) {
            console.error('获取环节数据失败:', error);
        } finally {
            setLoading(false);
        }
    }, [shipmentId]);

    // 加载停留记录
    const loadStopRecords = useCallback(async () => {
        if (!shipmentId) return;
        try {
            const response = await api.getShipmentStops(shipmentId, 1, 20) as any;
            let records: DeviceStopRecord[] = [];
            if (response?.data?.data?.records) {
                records = response.data.data.records;
            } else if (response?.data?.records) {
                records = response.data.records;
            } else if (response?.records) {
                records = response.records;
            }
            // 按开始时间倒序排列（最新的在前）
            setStopRecords(records.sort((a, b) =>
                new Date(b.start_time).getTime() - new Date(a.start_time).getTime()
            ));
        } catch (error) {
            console.error('获取停留记录失败:', error);
            setStopRecords([]);
        }
    }, [shipmentId]);

    // 初始加载
    useEffect(() => {
        setLoading(true);
        loadStages();
        loadStopRecords();
    }, [shipmentId, loadStages, loadStopRecords]);

    // 轮询更新
    useEffect(() => {
        if (!shipmentId || refreshInterval <= 0) return;

        const interval = setInterval(() => {
            loadStages();
            loadStopRecords();
        }, refreshInterval);

        return () => clearInterval(interval);
    }, [shipmentId, refreshInterval, loadStages, loadStopRecords]);

    // 活跃停留本地时钟，确保停留时长实时增长
    useEffect(() => {
        const timer = setInterval(() => setNowTs(Date.now()), 1000);
        return () => clearInterval(timer);
    }, []);

    // 判断是否为国内运输
    const isDomesticShipment = (): boolean => {
        // 如果有明确的起运港/目的港/中转港等，一定是跨境
        const hasCrossBorderStages = stages.some(s =>
            s.stage_code === 'origin_port' ||
            s.stage_code === 'dest_port' ||
            s.stage_code === 'transit_port' ||
            s.stage_code === 'main_line'
        );
        if (hasCrossBorderStages) return false;

        return true;
    };

    // 合并港口节点（跨境运输用）
    const mergePortNodes = (stages: ShipmentStage[]): ShipmentStage[] => {
        const merged: ShipmentStage[] = [];

        for (let i = 0; i < stages.length; i++) {
            const current = stages[i];
            const next = stages[i + 1];

            // 检查是否可以与下一个节点合并
            if (next) {
                // 起运港合并
                if (current.stage_code === 'origin_arrival' && next.stage_code === 'origin_departure') {
                    merged.push({
                        ...current,
                        id: `merged-origin-${current.id}`,
                        stage_name: '起运港',
                        stage_code: 'origin_port' as any,
                        // 状态：两个都完成才算完成，任何一个进行中就是进行中
                        status: (current.status === 'completed' && next.status === 'completed') ? 'completed' :
                            (current.status === 'in_progress' || next.status === 'in_progress') ? 'in_progress' :
                                'pending' as StageStatus,
                        trigger_type: next.trigger_type || current.trigger_type,
                        actual_start: current.actual_start,
                        actual_end: next.actual_end,
                    });
                    i++; // 跳过下一个
                    continue;
                }

                // 中转港合并
                if (current.stage_code === 'transit_arrival' && next.stage_code === 'transit_departure') {
                    merged.push({
                        ...current,
                        id: `merged-transit-${current.id}`,
                        stage_name: '中转港',
                        stage_code: 'transit_port' as any,
                        status: (current.status === 'completed' && next.status === 'completed') ? 'completed' :
                            (current.status === 'in_progress' || next.status === 'in_progress') ? 'in_progress' :
                                'pending' as StageStatus,
                        trigger_type: next.trigger_type || current.trigger_type,
                        actual_start: current.actual_start,
                        actual_end: next.actual_end,
                    });
                    i++;
                    continue;
                }

                // 目的港合并
                if (current.stage_code === 'dest_arrival' && next.stage_code === 'dest_departure') {
                    merged.push({
                        ...current,
                        id: `merged-dest-${current.id}`,
                        stage_name: '目的港',
                        stage_code: 'dest_port' as any,
                        status: (current.status === 'completed' && next.status === 'completed') ? 'completed' :
                            (current.status === 'in_progress' || next.status === 'in_progress') ? 'in_progress' :
                                'pending' as StageStatus,
                        trigger_type: next.trigger_type || current.trigger_type,
                        actual_start: current.actual_start,
                        actual_end: next.actual_end,
                    });
                    i++;
                    continue;
                }
            }

            // 不能合并的节点直接保留
            merged.push(current);
        }

        return merged;
    };

    // 简化为国内运输节点
    const simplifyToDomesticNodes = (stages: ShipmentStage[]): ShipmentStage[] => {
        const simplified: ShipmentStage[] = [];

        // 查找各个关键节点
        const preTransit = stages.find(s => s.stage_code === 'pre_transit' || s.stage_code === 'pickup' || s.stage_code === 'first_mile');
        const delivery = stages.find(s => s.stage_code === 'delivery' || s.stage_code === 'last_mile');
        const delivered = stages.find(s => s.stage_code === 'delivered');

        // 1. 前程运输
        if (preTransit) {
            simplified.push({
                ...preTransit,
                stage_name: preTransit.status === 'completed' ? '前程运输完成' : '前程运输',
            });
        }

        // 2. 运输中
        if (delivery) {
            if (delivery.status === 'in_progress') {
                simplified.push({
                    ...delivery,
                    stage_name: '运输中',
                });
            } else if (delivery.status === 'completed') {
                // 如果delivery已完成，显示"已到达"
                simplified.push({
                    ...delivery,
                    stage_name: '已到达',
                });
            } else {
                // pending状态
                simplified.push({
                    ...delivery,
                    stage_name: '运输中',
                });
            }
        }

        // 3. 签收
        if (delivered) {
            simplified.push({
                ...delivered,
                stage_name: '签收',
            });
        }

        return simplified;
    };

    // 格式化时间
    const formatTime = (dateStr: string | undefined) => {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        return `${date.getMonth() + 1}/${date.getDate()}`;
    };

    // 格式化完整时间
    const formatFullTime = (dateStr: string | undefined) => {
        if (!dateStr) return '';
        return new Date(dateStr).toLocaleString('zh-CN');
    };

    // 获取状态图标
    const getStatusIcon = (status: StageStatus) => {
        switch (status) {
            case 'completed':
                return <CheckCircleFilled style={{ color: '#52c41a' }} />;
            case 'in_progress':
                return <ClockCircleFilled style={{ color: '#1890ff' }} />;
            case 'skipped':
                return <MinusCircleFilled style={{ color: '#faad14' }} />;
            default:
                return null;
        }
    };

    const formatDurationText = (seconds: number): string => {
        if (seconds < 60) return `${seconds}秒`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`;
        if (seconds < 86400) {
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`;
        }
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        return hours > 0 ? `${days}天${hours}小时` : `${days}天`;
    };

    const getLiveDurationSeconds = (record: DeviceStopRecord): number => {
        if (record.status !== 'active') {
            return record.duration_seconds || 0;
        }
        const startMs = new Date(record.start_time).getTime();
        if (!Number.isFinite(startMs)) {
            return record.duration_seconds || 0;
        }
        return Math.max(0, Math.floor((nowTs - startMs) / 1000));
    };

    const getLiveDurationText = (record: DeviceStopRecord): string => {
        if (record.status !== 'active') {
            return record.duration_text || formatDurationText(record.duration_seconds || 0);
        }
        return formatDurationText(getLiveDurationSeconds(record));
    };

    const getRecordStartTs = (record: DeviceStopRecord): number => {
        const ts = new Date(record.start_time).getTime();
        return Number.isFinite(ts) ? ts : 0;
    };

    const getRecordCreatedTs = (record: DeviceStopRecord): number => {
        const ts = new Date(record.created_at).getTime();
        return Number.isFinite(ts) ? ts : 0;
    };

    // 获取停留时长标签颜色
    const getStopDurationColor = (durationSeconds: number, status: string) => {
        if (status === 'active') {
            return 'processing';
        }
        const hours = durationSeconds / 3600;
        if (hours >= 24) return 'red';
        if (hours >= 12) return 'orange';
        if (hours >= 6) return 'gold';
        return 'green';
    };

    // 处理显示节点：根据类型选择合并或简化
    let processedStages = stages.length > 0 ? stages : DEFAULT_STAGES.map((s, i) => ({
        id: `default-${i}`,
        shipment_id: shipmentId,
        stage_code: s.code as any,
        stage_name: s.name,
        stage_icon: StageIcons[s.code] || '📋',
        stage_order: i + 1,
        status: 'pending' as StageStatus,
        cost: 0,
        currency: 'CNY',
        actual_start: undefined,
        actual_end: undefined,
        trigger_type: undefined,
    } as ShipmentStage));

    // 根据运单类型处理节点
    if (stages.length > 0) {
        if (isDomesticShipment()) {
            // 国内运输：简化为4个节点
            processedStages = simplifyToDomesticNodes(stages);
        } else {
            // 跨境运输：如果有分开的到达/离开节点则合并，否则就是标准的(如 pre_transit, origin_port 等)全显示
            const hasLegacyPortNodes = stages.some(s => s.stage_code.includes('arrival') || s.stage_code.includes('departure'));
            if (hasLegacyPortNodes) {
                processedStages = mergePortNodes(stages);
            } else {
                processedStages = stages;
            }
        }
    }

    // 如果没有后端数据，使用默认展示
    const displayStages = processedStages;
    const latestStopId = stopRecords.length > 0
        ? [...stopRecords]
            .sort((a, b) => {
                const byStart = getRecordStartTs(b) - getRecordStartTs(a);
                if (byStart !== 0) return byStart;
                return getRecordCreatedTs(b) - getRecordCreatedTs(a);
            })[0]?.id
        : undefined;

    if (loading && stages.length === 0) {
        return (
            <div className="transport-timeline-loading">
                <Spin size="small" />
            </div>
        );
    }

    // 构建时间线元素数组（包含运输节点和停留点）
    const timelineElements: Array<{ type: 'stage' | 'stop'; data: any; index: number }> = [];

    // 添加运输节点
    displayStages.forEach((stage, index) => {
        timelineElements.push({ type: 'stage', data: stage, index });
    });

    return (
        <div className={`transport-node-timeline ${compact ? 'compact' : ''}`}>
            {/* 运输节点 */}
            {displayStages.map((stage, index) => {
                const isCompleted = stage.status === 'completed';
                const isInProgress = stage.status === 'in_progress';
                const isSkipped = stage.status === 'skipped';
                const icon = StageIcons[stage.stage_code] || '📋';
                const triggerType = (stage as any).trigger_type || '';

                return (
                    <React.Fragment key={stage.id || `stage-${index}`}>
                        {/* 节点 */}
                        <Tooltip
                            title={
                                <div className="node-tooltip">
                                    <div><strong>{stage.stage_name}</strong></div>
                                    {stage.actual_start && (
                                        <div>开始: {formatFullTime(stage.actual_start)}</div>
                                    )}
                                    {stage.actual_end && (
                                        <div>完成: {formatFullTime(stage.actual_end)}</div>
                                    )}
                                    {triggerType && (
                                        <div>触发: {TriggerLabels[triggerType] || triggerType}</div>
                                    )}
                                </div>
                            }
                        >
                            <div
                                className={`timeline-node ${isCompleted ? 'completed' :
                                    isInProgress ? 'in-progress' :
                                        isSkipped ? 'skipped' : 'pending'
                                    }`}
                            >
                                <div
                                    className="node-icon-wrapper"
                                    style={{ borderColor: StatusColors[stage.status] }}
                                >
                                    <span className="node-icon">{icon}</span>
                                    {isInProgress && <div className="pulse-ring" />}
                                </div>

                                <div className="node-label">{stage.stage_name}</div>

                                {!compact && (
                                    <>
                                        {/* 触发方式 */}
                                        {isCompleted && triggerType && (
                                            <div className="node-trigger">
                                                {TriggerIcons[triggerType]}
                                                <span>{TriggerLabels[triggerType] || triggerType}</span>
                                            </div>
                                        )}

                                        {/* 节点时间：已完成显示完成时间，进行中显示开始时间 */}
                                        {((isCompleted && stage.actual_end) || (isInProgress && (stage.actual_start || (stage as any).updated_at))) && (
                                            <div className="node-time">
                                                {isCompleted
                                                    ? formatTime(stage.actual_end)
                                                    : formatTime(stage.actual_start || (stage as any).updated_at)}
                                            </div>
                                        )}
                                    </>
                                )}

                                {/* 状态指示器 */}
                                <div className="node-status-indicator">
                                    {getStatusIcon(stage.status) || (
                                        <span className="status-dot" style={{ background: StatusColors[stage.status] }} />
                                    )}
                                </div>
                            </div>
                        </Tooltip>

                        {/* 连接线 */}
                        {index < displayStages.length - 1 && (
                            <div
                                className={`timeline-connector ${isCompleted ? 'completed' : ''}`}
                            />
                        )}
                    </React.Fragment>
                );
            })}

            {/* 停留点标记 - 显示在时间线下方 */}
            {!compact && stopRecords.length > 0 && (
                <div className="timeline-stop-markers">
                    {stopRecords.map((record) => {
                        const isLatestStop = record.id === latestStopId;
                        const isActiveStop = record.status === 'active';
                        return (
                        <Tooltip
                            key={record.id}
                            title={
                                <div className="stop-tooltip">
                                    <div><strong>停留点</strong></div>
                                    <div>开始: {formatFullTime(record.start_time)}</div>
                                    {record.end_time && <div>结束: {formatFullTime(record.end_time)}</div>}
                                    <div>时长: {getLiveDurationText(record)}</div>
                                    {record.address && (
                                        <div className="stop-address">
                                            <EnvironmentOutlined style={{ marginRight: 4 }} />
                                            {record.address}
                                        </div>
                                    )}
                                </div>
                            }
                        >
                            <div className={`stop-marker ${isLatestStop ? 'active' : ''}`}>
                                <StopOutlined className="stop-icon" />
                                <Tag
                                    color={getStopDurationColor(getLiveDurationSeconds(record), record.status)}
                                    className="stop-duration-tag"
                                >
                                    {getLiveDurationText(record)}
                                </Tag>
                                {isLatestStop && isActiveStop && <div className="stop-pulse" />}
                            </div>
                        </Tooltip>
                        );
                    })}
                </div>
            )}
        </div>
    );
};

export default TransportNodeTimeline;
