import React, { useEffect, useState } from 'react';
import { Switch, Space, message, Divider } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import api from '../api/client';

interface ShipmentFieldConfig {
    bill_of_lading: boolean;
    container_no: boolean;
    seal_no: boolean;
    vessel_name: boolean;
    voyage_no: boolean;
    carrier: boolean;
    po_numbers: boolean;
    sku_ids: boolean;
    fba_shipment_id: boolean;
    surcharges: boolean;
    customs_fee: boolean;
    other_cost: boolean;
}

const ShipmentFieldSettings: React.FC = () => {
    const [config, setConfig] = useState<ShipmentFieldConfig>({
        bill_of_lading: false,
        container_no: false,
        seal_no: false,
        vessel_name: false,
        voyage_no: false,
        carrier: false,
        po_numbers: false,
        sku_ids: false,
        fba_shipment_id: false,
        surcharges: false,
        customs_fee: false,
        other_cost: false,
    });
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = async () => {
        setLoading(true);
        try {
            const res = await api.getShipmentFieldConfig();
            if (res.data) {
                setConfig(res.data);
            }
        } catch (error: any) {
            const errorMsg = error?.response?.data?.error || error?.message || '加载配置失败';
            console.error('加载运单字段配置失败:', error);
            if (error?.response?.status === 401) {
                message.error('请先登录');
            } else if (error?.response?.status === 403) {
                message.error('权限不足，仅管理员可访问');
            } else {
                message.error('加载配置失败: ' + errorMsg);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            await api.updateShipmentFieldConfig(config);
            message.success('运单字段配置已保存');
        } catch (error) {
            message.error('保存配置失败');
        } finally {
            setSaving(false);
        }
    };

    return (
        <div>
            <div style={{ marginBottom: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                    <div style={{ color: '#1890ff', fontSize: 14, fontWeight: 500 }}>
                    ℹ️ 功能说明
                    </div>
                    <div style={{ color: '#666', fontSize: 13 }}>
                        开启字段后，创建运单时会显示对应字段；关闭后则隐藏。所有字段默认关闭。
                    </div>
                </Space>
            </div>

            {/* 📦 货物与单证 */}
            <Divider orientation={"left" as any}>📦 货物与单证</Divider>
            <div style={{ marginBottom: 24 }}>
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
                    <div style={{ width: 150, fontSize: 14 }}>提单号</div>
                    <Space>
                        <Switch
                            checked={config.bill_of_lading}
                            onChange={(checked) => setConfig({ ...config, bill_of_lading: checked })}
                            disabled={loading}
                        />
                        <span style={{ color: '#999', fontSize: 12 }}>
                            {config.bill_of_lading ? '已开启' : '已关闭'}
                        </span>
                        <span style={{ color: '#bbb', fontSize: 12 }}>(MBL/HBL/AWB)</span>
                    </Space>
                </div>
            </div>

            {/* 🚢 单证船务 */}
            <Divider orientation={"left" as any}>🚢 单证船务</Divider>
            <div style={{ marginBottom: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>箱号/车牌</div>
                        <Space>
                            <Switch
                                checked={config.container_no}
                                onChange={(checked) => setConfig({ ...config, container_no: checked })}
                                disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.container_no ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>封条号</div>
                        <Space>
                            <Switch
                                    checked={config.seal_no}
                                    onChange={(checked) => setConfig({ ...config, seal_no: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.seal_no ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>船名</div>
                        <Space>
                            <Switch
                                checked={config.vessel_name}
                                onChange={(checked) => setConfig({ ...config, vessel_name: checked })}
                                disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.vessel_name ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>航次</div>
                        <Space>
                            <Switch
                                    checked={config.voyage_no}
                                    onChange={(checked) => setConfig({ ...config, voyage_no: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.voyage_no ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>船司/航司</div>
                        <Space>
                            <Switch
                                    checked={config.carrier}
                                    onChange={(checked) => setConfig({ ...config, carrier: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.carrier ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>
                </Space>
            </div>

            {/* 📋 订单关联 */}
            <Divider orientation={"left" as any}>📋 订单关联</Divider>
            <div style={{ marginBottom: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>PO单号</div>
                        <Space>
                            <Switch
                                    checked={config.po_numbers}
                                    onChange={(checked) => setConfig({ ...config, po_numbers: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.po_numbers ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>SKU ID</div>
                        <Space>
                            <Switch
                                    checked={config.sku_ids}
                                    onChange={(checked) => setConfig({ ...config, sku_ids: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.sku_ids ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>FBA发货单号</div>
                        <Space>
                            <Switch
                                    checked={config.fba_shipment_id}
                                    onChange={(checked) => setConfig({ ...config, fba_shipment_id: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.fba_shipment_id ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>
                </Space>
            </div>

            {/* 💰 费用信息 */}
            <Divider orientation={"left" as any}>💰 费用信息</Divider>
            <div style={{ marginBottom: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                    <div style={{ fontSize: 13, color: '#666', marginBottom: 16 }}>
                        <span>⚠️ 运费始终显示，不可关闭</span>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>附加费</div>
                        <Space>
                            <Switch
                                    checked={config.surcharges}
                                    onChange={(checked) => setConfig({ ...config, surcharges: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.surcharges ? '已开启' : '已关闭'}
                            </span>
                            <span style={{ color: '#bbb', fontSize: 12 }}>(关闭后只显示运费)</span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>关税</div>
                        <Space>
                            <Switch
                                    checked={config.customs_fee}
                                    onChange={(checked) => setConfig({ ...config, customs_fee: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.customs_fee ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center' }}>
                        <div style={{ width: 150 }}>其他费用</div>
                        <Space>
                            <Switch
                                    checked={config.other_cost}
                                    onChange={(checked) => setConfig({ ...config, other_cost: checked })}
                                    disabled={loading}
                            />
                            <span style={{ color: '#999', fontSize: 12 }}>
                                {config.other_cost ? '已开启' : '已关闭'}
                            </span>
                        </Space>
                    </div>
                </Space>
            </div>

            {/* 保存按钮 */}
            <div style={{ marginTop: 24, paddingTop: 16, borderTop: '1px solid #f0f0f0' }}>
                <button
                    onClick={handleSave}
                    disabled={loading || saving}
                    style={{
                        padding: '8px 24px',
                        background: saving ? '#d9d9d9' : '#1890ff',
                        color: 'white',
                        border: 'none',
                        borderRadius: 4,
                        cursor: loading || saving ? 'not-allowed' : 'pointer',
                        fontSize: 14,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8
                    }}
                >
                    <SaveOutlined />
                    {saving ? '保存中...' : '保存配置'}
                </button>
            </div>
        </div>
    );
};

export default ShipmentFieldSettings;
