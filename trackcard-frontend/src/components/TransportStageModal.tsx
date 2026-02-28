import React, { useState, useEffect } from 'react';
import { Modal, Steps, Card, Tag, Button, message, Spin, Descriptions, Space, Row, Col, Divider, Empty, Progress, Input, InputNumber, Tooltip } from 'antd';
import {
    TruckOutlined,
    RocketOutlined,
    GlobalOutlined,
    BankOutlined,
    HomeOutlined,
    ArrowRightOutlined,
    InfoCircleOutlined,
    CheckCircleOutlined,
    SaveOutlined
} from '@ant-design/icons';
import api from '../api/client';
import type { ShipmentStage, StageCode, StageStatus, Shipment } from '../types';
import { useCurrencyStore } from '../store/currencyStore';
import './TransportStageModal.css';

interface TransportStageModalProps {
    shipment: Shipment | null;
    visible: boolean;
    onClose: () => void;
    onStageChange?: () => void;
}

// 环节图标映射
const StageIcons: Partial<Record<StageCode, React.ReactNode>> = {
    // New Standard Stages
    pre_transit: <TruckOutlined />,
    main_line: <GlobalOutlined />,
    transit_port: <GlobalOutlined />,
    delivered: <CheckCircleOutlined />,

    // Compatibility
    pickup: <TruckOutlined />,
    origin_arrival: <RocketOutlined />,
    origin_departure: <RocketOutlined />,
    transit_arrival: <GlobalOutlined />,
    transit_departure: <GlobalOutlined />,
    dest_arrival: <BankOutlined />,
    dest_departure: <BankOutlined />,
    delivery: <TruckOutlined />,

    // Legacy
    first_mile: <TruckOutlined />,
    origin_port: <RocketOutlined />,
    main_carriage: <GlobalOutlined />,
    dest_port: <BankOutlined />,
    last_mile: <HomeOutlined />,
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

// 环节状态中文
const StageStatusLabels: Record<StageStatus, string> = {
    pending: '待开始',
    in_progress: '进行中',
    completed: '已完成',
    skipped: '已跳过',
};

const TransportStageModal: React.FC<TransportStageModalProps> = ({
    shipment,
    visible,
    onClose,
    onStageChange,
}) => {
    const [loading, setLoading] = useState(true);
    const [stages, setStages] = useState<ShipmentStage[]>([]);
    const [_currentStage, setCurrentStage] = useState<StageCode>('first_mile');
    const [totalCost, setTotalCost] = useState(0);
    const [transitioning, setTransitioning] = useState(false);
    const [selectedStage, setSelectedStage] = useState<ShipmentStage | null>(null);
    const [signerName, setSignerName] = useState('');
    const [editingCost, setEditingCost] = useState<number | null>(null);
    const [savingCost, setSavingCost] = useState(false);

    const { getCurrencyConfig } = useCurrencyStore();
    const currencyConfig = getCurrencyConfig();

    // 加载环节数据
    const loadStages = async () => {
        if (!shipment?.id) return;
        try {
            setLoading(true);
            const response = await api.getShipmentStages(shipment.id);
            const data = response as any;
            // API returns { success: true, data: { stages: [...], current_stage: string, total_cost: number } }
            const stagesData = data?.data || data;
            if (stagesData) {
                setStages(stagesData.stages || []);
                setCurrentStage((stagesData.current_stage || 'first_mile') as StageCode);
                setTotalCost(stagesData.total_cost || 0);
                // 自动选中当前进行中的环节
                const inProgressStage = (stagesData.stages || []).find((s: ShipmentStage) => s.status === 'in_progress');
                setSelectedStage(inProgressStage || (stagesData.stages?.[0] || null));
            }
        } catch (error) {
            console.error('获取环节数据失败:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (visible && shipment?.id) {
            loadStages();
            // 重置签收人
            setSignerName('');
        }
    }, [visible, shipment?.id]);

    // 获取当前进行中的环节索引
    const getCurrentStepIndex = () => {
        const index = stages.findIndex(s => s.status === 'in_progress');
        if (index >= 0) return index;
        const completedCount = stages.filter(s => s.status === 'completed').length;
        return completedCount > 0 ? completedCount - 1 : 0;
    };

    // 推进到下一环节
    const handleTransition = async () => {
        if (!shipment?.id) return;

        // 如果是签收环节（最后一步），校验签收人
        const currentIsDelivered = stages.find(s => s.status === 'in_progress')?.stage_code === 'delivered';
        if (currentIsDelivered && !signerName.trim()) {
            message.warning('请输入签收人姓名');
            return;
        }

        try {
            setTransitioning(true);
            const note = currentIsDelivered ? `手动推进 (签收人: ${signerName})` : '手动推进';
            await api.transitionShipmentStage(shipment.id, note);
            message.success('环节已成功推进!');
            await loadStages();
            onStageChange?.();
        } catch (error: any) {
            message.error(error.response?.data?.error || '推进失败');
        } finally {
            setTransitioning(false);
        }
    };

    // 保存费用修改
    const handleSaveCost = async () => {
        if (!selectedStage || editingCost === null || !shipment) return;
        try {
            setSavingCost(true);
            await api.updateShipmentStage(shipment.id, selectedStage.stage_code, { cost: editingCost });
            message.success('费用保存成功');
            await loadStages();
            setEditingCost(null);
        } catch (error: any) {
            message.error(error.response?.data?.error || '保存失败');
        } finally {
            setSavingCost(false);
        }
    };

    // 判断是否所有环节都已完成
    const isAllCompleted = stages.length > 0 && stages.every(s => s.status === 'completed' || s.status === 'skipped');

    // 获取下一个待推进的环节名称
    const getNextStageName = () => {
        const inProgress = stages.find(s => s.status === 'in_progress');
        if (inProgress) {
            const idx = stages.findIndex(s => s.stage_code === inProgress.stage_code);
            return stages[idx + 1]?.stage_name || '完成';
        }
        const firstPending = stages.find(s => s.status === 'pending');
        return firstPending?.stage_name || '完成';
    };

    // check if current stage is delivered
    const currentInProgress = stages.find(s => s.status === 'in_progress');
    const isDeliveredStage = currentInProgress?.stage_code === 'delivered';

    return (
        <Modal
            title={
                <div className="stage-modal-title">
                    <GlobalOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                    运输环节管理
                    {shipment && <Tag color="blue" style={{ marginLeft: 12 }}>{shipment.id}</Tag>}
                </div>
            }
            open={visible}
            onCancel={onClose}
            width={900}
            footer={null}
            centered
            className="transport-stage-modal"
        >
            {loading ? (
                <div className="stage-loading">
                    <Spin size="large" tip="加载环节数据..." />
                </div>
            ) : stages.length === 0 ? (
                <Empty description="暂无环节数据" />
            ) : (
                <div className="stage-content">
                    {/* 顶部进度条 */}
                    <div className="stage-progress-section">
                        <Steps
                            current={getCurrentStepIndex()}
                            size="small"
                            items={stages.map(stage => ({
                                title: (
                                    <span
                                        className={`stage-step-title ${selectedStage?.stage_code === stage.stage_code ? 'active' : ''}`}
                                        onClick={() => setSelectedStage(stage)}
                                    >
                                        {stage.stage_name}
                                    </span>
                                ),
                                icon: (
                                    <div
                                        className={`stage-step-icon ${selectedStage?.stage_code === stage.stage_code ? 'selected' : ''}`}
                                        style={{ backgroundColor: StageStatusColors[stage.status] }}
                                        onClick={() => setSelectedStage(stage)}
                                    >
                                        {StageIcons[stage.stage_code] || <GlobalOutlined />}
                                    </div>
                                ),
                                status: stage.status === 'completed' ? 'finish' :
                                    stage.status === 'in_progress' ? 'process' : 'wait',
                            }))}
                        />
                    </div>

                    <Divider style={{ margin: '16px 0' }} />

                    {/* 操作区域 + 环节详情 */}
                    <Row gutter={24}>
                        {/* 左侧：快捷操作 */}
                        <Col span={8}>
                            <Card title="快捷操作" size="small" className="action-card">
                                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                                    <Button
                                        type="primary"
                                        icon={<ArrowRightOutlined />}
                                        block
                                        size="large"
                                        onClick={handleTransition}
                                        loading={transitioning}
                                        disabled={isAllCompleted}
                                    >
                                        {isAllCompleted ? '已全部完成' : `推进到: ${getNextStageName()}`}
                                    </Button>

                                    {/* 签收人输入框 - 仅在签收环节显示 */}
                                    {isDeliveredStage && (
                                        <div style={{ marginBottom: 16 }}>
                                            <div className="label" style={{ marginBottom: 8 }}>签收人</div>
                                            <Input
                                                placeholder="请输入签收人姓名"
                                                value={signerName}
                                                onChange={e => setSignerName(e.target.value)}
                                            />
                                        </div>
                                    )}

                                    <div className="stage-summary">
                                        {/* 新增: 运输进度展示 */}
                                        <div className="summary-item" style={{ marginBottom: 16 }}>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                                                <span className="label">运输进度</span>
                                                <Tag color={shipment?.status === 'delivered' ? 'success' : shipment?.status === 'in_transit' ? 'processing' : 'default'}>
                                                    {shipment?.status === 'pending' ? '待发货' :
                                                        shipment?.status === 'in_transit' ? '运输中' :
                                                            shipment?.status === 'delivered' ? '已送达' :
                                                                shipment?.status === 'cancelled' ? '已取消' : shipment?.status}
                                                </Tag>
                                            </div>
                                            <Progress
                                                percent={shipment?.progress || 0}
                                                status={shipment?.status === 'delivered' ? 'success' : 'active'}
                                                strokeColor={{
                                                    '0%': '#108ee9',
                                                    '100%': '#87d068',
                                                }}
                                            />
                                        </div>
                                        <div className="summary-item">
                                            <span className="label">当前环节</span>
                                            <span className="value">
                                                {stages.find(s => s.status === 'in_progress')?.stage_name || '-'}
                                            </span>
                                        </div>
                                        <div className="summary-item">
                                            <span className="label">完成进度</span>
                                            <span className="value">
                                                {stages.filter(s => s.status === 'completed').length} / {stages.length}
                                            </span>
                                        </div>
                                        <div className="summary-item">
                                            <span className="label">总费用</span>
                                            <span className="value cost">
                                                {currencyConfig.symbol}{totalCost.toFixed(2)}
                                            </span>
                                        </div>
                                    </div>
                                </Space>
                            </Card>
                        </Col>

                        {/* 右侧：环节详情 */}
                        <Col span={16}>
                            <Card
                                title={
                                    <span>
                                        {selectedStage ? (
                                            <>
                                                {StageIcons[selectedStage.stage_code] || <GlobalOutlined />}
                                                <span style={{ marginLeft: 8 }}>{selectedStage.stage_name}</span>
                                                <Tag color={StageStatusTagColors[selectedStage.status]} style={{ marginLeft: 8 }}>
                                                    {StageStatusLabels[selectedStage.status]}
                                                </Tag>
                                            </>
                                        ) : '选择环节查看详情'}
                                    </span>
                                }
                                size="small"
                                className="detail-card"
                            >
                                {selectedStage ? (
                                    <Descriptions column={2} size="small" bordered>
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
                                        <Descriptions.Item label={selectedStage.cost_name || '环节费用'}>
                                            <Space>
                                                <InputNumber
                                                    value={editingCost !== null ? editingCost : selectedStage.cost}
                                                    onChange={(val) => setEditingCost(val as number)}
                                                    precision={2}
                                                    min={0}
                                                    style={{ width: 120 }}
                                                    addonBefore={currencyConfig.symbol}
                                                />
                                                <Tooltip title="保存费用">
                                                    <Button
                                                        type="primary"
                                                        size="small"
                                                        icon={<SaveOutlined />}
                                                        onClick={handleSaveCost}
                                                        loading={savingCost}
                                                        disabled={editingCost === null || editingCost === selectedStage.cost}
                                                    />
                                                </Tooltip>
                                            </Space>
                                        </Descriptions.Item>
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
                                        {selectedStage.trigger_type && (
                                            <Descriptions.Item label="触发方式" span={2}>
                                                <Tag>
                                                    {selectedStage.trigger_type === 'manual' ? '手动操作' :
                                                        selectedStage.trigger_type === 'geofence' ? '电子围栏' : 'API回传'}
                                                </Tag>
                                            </Descriptions.Item>
                                        )}
                                    </Descriptions>
                                ) : (
                                    <Empty description="点击上方环节查看详情" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                                )}
                            </Card>
                        </Col>
                    </Row>

                    {/* 底部提示 */}
                    <div className="stage-footer">
                        <InfoCircleOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                        <span>点击环节图标可查看详情，推进操作将记录在运单日志中</span>
                    </div>
                </div>
            )}
        </Modal>
    );
};

export default TransportStageModal;
