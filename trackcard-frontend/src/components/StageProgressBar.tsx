
import React, { useState, useEffect } from 'react';
import { Steps, Card, Tag, Button, Tooltip, Modal, message, Spin, Descriptions } from 'antd';
import {
    TruckOutlined,
    RocketOutlined,
    GlobalOutlined,
    BankOutlined,
    HomeOutlined,
    ArrowRightOutlined,
    CheckCircleOutlined
} from '@ant-design/icons';
import api from '../api/client';
import type { ShipmentStage, StageCode, StageStatus } from '../types';
import { useCurrencyStore } from '../store/currencyStore';

interface StageProgressBarProps {
    shipmentId: string;
    compact?: boolean;         // 紧凑模式（仅显示进度条）
    showActions?: boolean;     // 是否显示操作按钮
    onStageChange?: () => void; // 环节变更回调
}

// 环节图标映射 - 使用 Partial 允许只定义部分阶段的图标
const StageIcons: Partial<Record<StageCode, React.ReactNode>> = {
    // New Standard Stages
    pre_transit: <TruckOutlined />,
    main_line: <GlobalOutlined />,
    transit_port: <BankOutlined />,
    delivered: <CheckCircleOutlined />,

    // Compatibility
    first_mile: <TruckOutlined />,
    origin_port: <RocketOutlined />,
    main_carriage: <GlobalOutlined />,
    dest_port: <BankOutlined />,
    last_mile: <HomeOutlined />,
    delivery: <HomeOutlined />,
};

// 环节状态颜色
const StageStatusColors: Record<StageStatus, string> = {
    pending: '#d9d9d9',
    in_progress: '#1890ff',
    completed: '#52c41a',
    skipped: '#faad14',
};

// 环节状态标签颜色
const StageStatusTagColors: Record<StageStatus, string> = {
    pending: 'default',
    in_progress: 'processing',
    completed: 'success',
    skipped: 'warning',
};

const StageProgressBar: React.FC<StageProgressBarProps> = ({
    shipmentId,
    compact = false,
    showActions = true,
    onStageChange,
}) => {
    const [loading, setLoading] = useState(true);
    const [stages, setStages] = useState<ShipmentStage[]>([]);
    const [_currentStage, setCurrentStage] = useState<StageCode>('first_mile');
    const [totalCost, setTotalCost] = useState(0);
    const [transitionModalVisible, setTransitionModalVisible] = useState(false);
    const [selectedStage, setSelectedStage] = useState<ShipmentStage | null>(null);
    const [detailModalVisible, setDetailModalVisible] = useState(false);
    const [transitioning, setTransitioning] = useState(false);

    const { getCurrencyConfig } = useCurrencyStore();
    const currencyConfig = getCurrencyConfig();

    // 加载环节数据
    const loadStages = async () => {
        try {
            setLoading(true);
            const response = await api.getShipmentStages(shipmentId);
            // API直接返回 { stages, current_stage, total_cost } 格式
            // response 已经是 axios response.data，所以直接访问属性
            const data = response as any;
            if (data) {
                setStages(data.stages || []);
                setCurrentStage((data.current_stage || 'first_mile') as StageCode);
                setTotalCost(data.total_cost || 0);
            }
        } catch (error) {
            console.error('获取环节数据失败:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (shipmentId) {
            loadStages();
        }
    }, [shipmentId]);

    // 获取当前进行中的环节索引
    const getCurrentStepIndex = () => {
        const index = stages.findIndex(s => s.status === 'in_progress');
        if (index >= 0) return index;
        // 如果没有进行中的，返回第一个pending的前一个
        const pendingIndex = stages.findIndex(s => s.status === 'pending');
        return pendingIndex > 0 ? pendingIndex - 1 : 0;
    };

    // 推进到下一环节
    const handleTransition = async () => {
        try {
            setTransitioning(true);
            await api.transitionShipmentStage(shipmentId, '手动推进');
            message.success('环节已推进');
            await loadStages();
            setTransitionModalVisible(false);
            onStageChange?.();
        } catch (error: any) {
            message.error(error.response?.data?.error || '推进失败');
        } finally {
            setTransitioning(false);
        }
    };

    // 查看环节详情
    const handleViewDetail = (stage: ShipmentStage) => {
        setSelectedStage(stage);
        setDetailModalVisible(true);
    };

    // 获取环节状态图标


    if (loading) {
        return <Spin size="small" />;
    }

    // 紧凑模式 - 仅显示简单进度条
    if (compact) {
        return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                {stages.map((stage, index) => (
                    <React.Fragment key={stage.stage_code}>
                        <Tooltip title={`${stage.stage_name}: ${stage.status === 'completed' ? '已完成' : stage.status === 'in_progress' ? '进行中' : '待开始'} `}>
                            <div
                                style={{
                                    width: 28,
                                    height: 28,
                                    borderRadius: '50%',
                                    backgroundColor: StageStatusColors[stage.status],
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    color: stage.status === 'pending' ? '#999' : '#fff',
                                    fontSize: 12,
                                    cursor: 'pointer',
                                }}
                                onClick={() => handleViewDetail(stage)}
                            >
                                {stage.stage_icon || StageIcons[stage.stage_code]}
                            </div>
                        </Tooltip>
                        {index < stages.length - 1 && (
                            <div
                                style={{
                                    width: 20,
                                    height: 2,
                                    backgroundColor: stages[index + 1].status !== 'pending' ? '#52c41a' : '#d9d9d9',
                                }}
                            />
                        )}
                    </React.Fragment>
                ))}
            </div>
        );
    }

    // 完整模式 - 显示详细信息
    return (
        <Card
            title="运输环节"
            size="small"
            extra={
                showActions && (
                    <Button
                        type="primary"
                        size="small"
                        icon={<ArrowRightOutlined />}
                        onClick={() => setTransitionModalVisible(true)}
                        disabled={stages.every(s => s.status === 'completed')}
                    >
                        推进下一环节
                    </Button>
                )
            }
        >
            <Steps
                current={getCurrentStepIndex()}
                size="small"
                style={{ marginBottom: 16 }}
                items={stages.map(stage => ({
                    title: (
                        <span
                            style={{ cursor: 'pointer' }}
                            onClick={() => handleViewDetail(stage)}
                        >
                            {stage.stage_name}
                        </span>
                    ),
                    description: (
                        <div style={{ fontSize: 12 }}>
                            <Tag color={StageStatusTagColors[stage.status]} style={{ fontSize: 10 }}>
                                {stage.status === 'completed' ? '已完成' :
                                    stage.status === 'in_progress' ? '进行中' :
                                        stage.status === 'skipped' ? '已跳过' : '待开始'}
                            </Tag>
                            {stage.cost > 0 && (
                                <span style={{ color: '#999', marginLeft: 4 }}>
                                    {currencyConfig.symbol}{stage.cost.toFixed(2)}
                                </span>
                            )}
                        </div>
                    ),
                    icon: StageIcons[stage.stage_code],
                    status: stage.status === 'completed' ? 'finish' :
                        stage.status === 'in_progress' ? 'process' : 'wait',
                }))}
            />

            {totalCost > 0 && (
                <div style={{ textAlign: 'right', color: '#666', fontSize: 13 }}>
                    总费用: <strong style={{ color: '#1890ff' }}>{currencyConfig.symbol}{totalCost.toFixed(2)}</strong>
                </div>
            )}

            {/* 推进确认弹窗 */}
            <Modal
                title="推进到下一环节"
                open={transitionModalVisible}
                onOk={handleTransition}
                onCancel={() => setTransitionModalVisible(false)}
                confirmLoading={transitioning}
                okText="确认推进"
                cancelText="取消"
            >
                <p>确定要将当前运单推进到下一个运输环节吗？</p>
                <p style={{ color: '#999' }}>
                    当前环节: <strong>{stages.find(s => s.status === 'in_progress')?.stage_name || '无'}</strong>
                </p>
            </Modal>

            {/* 环节详情弹窗 */}
            <Modal
                title={`${selectedStage?.stage_name || ''} - 环节详情`}
                open={detailModalVisible}
                onCancel={() => setDetailModalVisible(false)}
                footer={null}
                width={500}
            >
                {selectedStage && (
                    <Descriptions column={1} size="small" bordered>
                        <Descriptions.Item label="环节状态">
                            <Tag color={StageStatusTagColors[selectedStage.status]}>
                                {selectedStage.status === 'completed' ? '已完成' :
                                    selectedStage.status === 'in_progress' ? '进行中' :
                                        selectedStage.status === 'skipped' ? '已跳过' : '待开始'}
                            </Tag>
                        </Descriptions.Item>
                        {selectedStage.partner_name && (
                            <Descriptions.Item label="负责方">
                                {selectedStage.partner_name}
                            </Descriptions.Item>
                        )}
                        {selectedStage.vehicle_plate && (
                            <Descriptions.Item label="拖车车牌">
                                {selectedStage.vehicle_plate}
                            </Descriptions.Item>
                        )}
                        {selectedStage.vessel_name && (
                            <Descriptions.Item label="船名">
                                {selectedStage.vessel_name}
                            </Descriptions.Item>
                        )}
                        {selectedStage.voyage_no && (
                            <Descriptions.Item label="航次">
                                {selectedStage.voyage_no}
                            </Descriptions.Item>
                        )}
                        {selectedStage.carrier && (
                            <Descriptions.Item label="船司/航司">
                                {selectedStage.carrier}
                            </Descriptions.Item>
                        )}
                        {selectedStage.actual_start && (
                            <Descriptions.Item label="实际开始">
                                {new Date(selectedStage.actual_start).toLocaleString('zh-CN')}
                            </Descriptions.Item>
                        )}
                        {selectedStage.actual_end && (
                            <Descriptions.Item label="实际结束">
                                {new Date(selectedStage.actual_end).toLocaleString('zh-CN')}
                            </Descriptions.Item>
                        )}
                        <Descriptions.Item label="环节费用">
                            {currencyConfig.symbol}{selectedStage.cost?.toFixed(2) || '0.00'}
                        </Descriptions.Item>
                        {selectedStage.trigger_type && (
                            <Descriptions.Item label="触发方式">
                                {selectedStage.trigger_type === 'manual' ? '手动操作' :
                                    selectedStage.trigger_type === 'geofence' ? '电子围栏' : 'API回传'}
                            </Descriptions.Item>
                        )}
                    </Descriptions>
                )}
            </Modal>
        </Card>
    );
};

export default StageProgressBar;
