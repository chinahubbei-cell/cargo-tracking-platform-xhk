import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Button, Space, Input, Select, Tag, Modal, Form, message, Row, Col, Tooltip, Popconfirm, InputNumber, DatePicker, Statistic, Badge, Divider } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, DollarOutlined, ReloadOutlined, ThunderboltOutlined, GlobalOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { api } from '../api/client';
import type { FreightRate, ContainerType, RateCreateRequest, RateCompareResult, RouteInfo, Partner } from '../types';
import { ContainerTypeNames } from '../types';
import { useCurrencyStore } from '../store/currencyStore';
import './Rates.css';

const { Option } = Select;
const { RangePicker } = DatePicker;

const Rates: React.FC = () => {
    const [rates, setRates] = useState<FreightRate[]>([]);
    const [partners, setPartners] = useState<Partner[]>([]);
    const [routes, setRoutes] = useState<RouteInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalVisible, setModalVisible] = useState(false);
    const [compareVisible, setCompareVisible] = useState(false);
    const [compareResults, setCompareResults] = useState<RateCompareResult[]>([]);
    const [compareLoading, setCompareLoading] = useState(false);
    const [editingRate, setEditingRate] = useState<FreightRate | null>(null);
    const [filters, setFilters] = useState<{ origin?: string; destination?: string; container_type?: string }>({});
    const [form] = Form.useForm();
    const [compareForm] = Form.useForm();
    const { formatAmount, getCurrencyConfig } = useCurrencyStore();
    const currencyConfig = getCurrencyConfig();

    const fetchRates = useCallback(async () => {
        setLoading(true);
        try {
            const response = await api.getRates({ ...filters, active: 'true' });
            if (response.success) {
                setRates(response.data || []);
            }
        } catch (error) {
            message.error('获取运价列表失败');
        } finally {
            setLoading(false);
        }
    }, [filters]);

    const fetchPartners = async () => {
        try {
            const response = await api.getPartners({ type: 'forwarder', status: 'active' });
            if (response.success) {
                setPartners(response.data || []);
            }
        } catch (error) {
            console.error('Failed to fetch partners');
        }
    };

    const fetchRoutes = async () => {
        try {
            const response = await api.getAvailableRoutes();
            if (response.success) {
                setRoutes(response.data || []);
            }
        } catch (error) {
            console.error('Failed to fetch routes');
        }
    };

    useEffect(() => {
        fetchRates();
        fetchPartners();
        fetchRoutes();
    }, [fetchRates]);

    const handleCreate = () => {
        setEditingRate(null);
        form.resetFields();
        setModalVisible(true);
    };

    const handleEdit = (rate: FreightRate) => {
        setEditingRate(rate);
        form.setFieldsValue({
            ...rate,
            valid_range: [dayjs(rate.valid_from), dayjs(rate.valid_to)],
        });
        setModalVisible(true);
    };

    const handleDelete = async (id: number) => {
        try {
            await api.deleteRate(id);
            message.success('删除成功');
            fetchRates();
        } catch (error) {
            message.error('删除失败');
        }
    };

    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            const [validFrom, validTo] = values.valid_range;

            const data: RateCreateRequest = {
                partner_id: values.partner_id,
                origin: values.origin,
                origin_name: values.origin_name,
                destination: values.destination,
                destination_name: values.destination_name,
                transit_days: values.transit_days,
                carrier: values.carrier,
                container_type: values.container_type,
                currency: values.currency || 'USD',
                ocean_freight: values.ocean_freight || 0,
                baf: values.baf || 0,
                caf: values.caf || 0,
                pss: values.pss || 0,
                gri: values.gri || 0,
                thc: values.thc || 0,
                doc_fee: values.doc_fee || 0,
                seal_fee: values.seal_fee || 0,
                other_fee: values.other_fee || 0,
                valid_from: validFrom.format('YYYY-MM-DDTHH:mm:ssZ'),
                valid_to: validTo.format('YYYY-MM-DDTHH:mm:ssZ'),
                remarks: values.remarks,
            };

            if (editingRate) {
                await api.updateRate(editingRate.id, data);
                message.success('更新成功');
            } else {
                await api.createRate(data);
                message.success('创建成功');
            }
            setModalVisible(false);
            fetchRates();
        } catch (error) {
            message.error('操作失败');
        }
    };

    const handleCompare = async () => {
        try {
            const values = await compareForm.validateFields();
            setCompareLoading(true);
            const response = await api.compareRates({
                origin: values.origin,
                destination: values.destination,
                container_type: values.container_type,
            });
            if (response.success && response.data) {
                setCompareResults(response.data.rates || []);
                if (response.data.rates?.length === 0) {
                    message.info('暂无符合条件的运价');
                }
            }
        } catch (error) {
            message.error('比价失败');
        } finally {
            setCompareLoading(false);
        }
    };

    const columns: ColumnsType<FreightRate> = [
        {
            title: '航线',
            key: 'route',
            width: 180,
            render: (_, record) => (
                <div>
                    <div className="route-text">{record.origin} → {record.destination}</div>
                    <div className="route-name">{record.origin_name} - {record.destination_name}</div>
                </div>
            ),
        },
        {
            title: '货代',
            dataIndex: 'partner_name',
            key: 'partner_name',
            width: 120,
        },
        {
            title: '船司',
            dataIndex: 'carrier',
            key: 'carrier',
            width: 100,
        },
        {
            title: '柜型',
            dataIndex: 'container_type',
            key: 'container_type',
            width: 100,
            render: (type: ContainerType) => (
                <Tag color="blue">{ContainerTypeNames[type] || type}</Tag>
            ),
        },
        {
            title: '航程',
            dataIndex: 'transit_days',
            key: 'transit_days',
            width: 80,
            align: 'center',
            render: (days: number) => days ? `${days}天` : '-',
        },
        {
            title: '海运费',
            dataIndex: 'ocean_freight',
            key: 'ocean_freight',
            width: 100,
            align: 'right',
            render: (val: number) => formatAmount(val),
        },
        {
            title: '总费用',
            dataIndex: 'total_fee',
            key: 'total_fee',
            width: 120,
            align: 'right',
            render: (val: number) => (
                <span className="total-fee">{formatAmount(val)}</span>
            ),
        },
        {
            title: '有效期',
            key: 'validity',
            width: 180,
            render: (_, record) => {
                const now = new Date();
                const validTo = new Date(record.valid_to);
                const daysLeft = Math.ceil((validTo.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
                return (
                    <div>
                        <div>{dayjs(record.valid_from).format('MM/DD')} - {dayjs(record.valid_to).format('MM/DD')}</div>
                        {daysLeft <= 7 && daysLeft > 0 && (
                            <Tag color="warning">即将到期</Tag>
                        )}
                    </div>
                );
            },
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_, record) => (
                <Space size="small">
                    <Tooltip title="编辑">
                        <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                    </Tooltip>
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
                        <Tooltip title="删除">
                            <Button type="text" danger icon={<DeleteOutlined />} />
                        </Tooltip>
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    const compareColumns: ColumnsType<RateCompareResult> = [
        {
            title: '排名',
            dataIndex: 'rank',
            key: 'rank',
            width: 60,
            render: (rank: number) => (
                <Badge count={rank} style={{ backgroundColor: rank === 1 ? '#52c41a' : rank <= 3 ? '#1890ff' : '#d9d9d9' }} />
            ),
        },
        {
            title: '货代',
            dataIndex: 'partner_name',
            key: 'partner_name',
            width: 120,
        },
        {
            title: '船司',
            dataIndex: 'carrier',
            key: 'carrier',
            width: 100,
        },
        {
            title: '航程',
            dataIndex: 'transit_days',
            key: 'transit_days',
            width: 80,
            render: (days: number) => days ? `${days}天` : '-',
        },
        {
            title: '总费用',
            dataIndex: 'total_fee',
            key: 'total_fee',
            width: 120,
            render: (val: number, record) => (
                <div>
                    <div className="compare-price">${val.toLocaleString()}</div>
                    {record.price_diff > 0 && (
                        <div className="price-diff">+${record.price_diff.toFixed(0)} ({record.price_diff_pct.toFixed(1)}%)</div>
                    )}
                </div>
            ),
        },
        {
            title: '推荐',
            dataIndex: 'recommendation',
            key: 'recommendation',
            width: 120,
            render: (rec: string) => rec ? <Tag color="green">{rec}</Tag> : null,
        },
    ];

    return (
        <div className="rates-page">
            <Row gutter={16}>
                {/* 左侧：航线概览 */}
                <Col span={6}>
                    <Card title={<Space><GlobalOutlined /> 航线概览</Space>} className="routes-card">
                        <div className="routes-list">
                            {routes.map((route, idx) => (
                                <div key={idx} className="route-item" onClick={() => {
                                    setFilters({ origin: route.origin, destination: route.destination });
                                }}>
                                    <div className="route-lane">{route.origin} → {route.destination}</div>
                                    <div className="route-stats">
                                        <span>{route.options_count}个报价</span>
                                        <span className="lowest">最低 {formatAmount(route.lowest_price)}</span>
                                    </div>
                                </div>
                            ))}
                            {routes.length === 0 && <div className="empty-text">暂无航线数据</div>}
                        </div>
                    </Card>
                </Col>

                {/* 右侧：运价列表 */}
                <Col span={18}>
                    <Card
                        title={<Space><DollarOutlined /> 运价管理</Space>}
                        extra={
                            <Space>
                                <Select
                                    placeholder="柜型"
                                    style={{ width: 120 }}
                                    allowClear
                                    onChange={(value) => setFilters({ ...filters, container_type: value })}
                                >
                                    <Option value="20GP">20GP</Option>
                                    <Option value="40GP">40GP</Option>
                                    <Option value="40HQ">40HQ</Option>
                                    <Option value="LCL">散货</Option>
                                </Select>
                                <Button icon={<ReloadOutlined />} onClick={fetchRates}>刷新</Button>
                                <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => setCompareVisible(true)}>智能比价</Button>
                                <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新增运价</Button>
                            </Space>
                        }
                    >
                        <Table
                            columns={columns}
                            dataSource={rates}
                            rowKey="id"
                            loading={loading}
                            pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
                            size="middle"
                        />
                    </Card>
                </Col>
            </Row>

            {/* 创建/编辑对话框 */}
            <Modal
                title={editingRate ? '编辑运价' : '新增运价'}
                open={modalVisible}
                onOk={handleSubmit}
                onCancel={() => setModalVisible(false)}
                width={800}
            >
                <Form form={form} layout="vertical">
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="partner_id" label="货代" rules={[{ required: true }]}>
                                <Select placeholder="选择货代">
                                    {partners.map(p => <Option key={p.id} value={p.id}>{p.name}</Option>)}
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="carrier" label="船司">
                                <Input placeholder="如 COSCO, MSC, Maersk" />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={6}>
                            <Form.Item name="origin" label="起运港代码" rules={[{ required: true }]}>
                                <Input placeholder="CNSHA" />
                            </Form.Item>
                        </Col>
                        <Col span={6}>
                            <Form.Item name="origin_name" label="起运港名称">
                                <Input placeholder="上海" />
                            </Form.Item>
                        </Col>
                        <Col span={6}>
                            <Form.Item name="destination" label="目的港代码" rules={[{ required: true }]}>
                                <Input placeholder="USLAX" />
                            </Form.Item>
                        </Col>
                        <Col span={6}>
                            <Form.Item name="destination_name" label="目的港名称">
                                <Input placeholder="洛杉矶" />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="container_type" label="柜型" rules={[{ required: true }]}>
                                <Select placeholder="选择柜型">
                                    <Option value="20GP">20GP</Option>
                                    <Option value="40GP">40GP</Option>
                                    <Option value="40HQ">40HQ</Option>
                                    <Option value="45HQ">45HQ</Option>
                                    <Option value="LCL">散货</Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="transit_days" label="航程(天)">
                                <InputNumber min={1} max={100} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="valid_range" label="有效期" rules={[{ required: true }]}>
                                <RangePicker style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Divider>费用明细 ({currencyConfig.code})</Divider>
                    <Row gutter={16}>
                        <Col span={6}><Form.Item name="ocean_freight" label="海运费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="baf" label="BAF燃油附加费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="thc" label="THC码头操作费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="doc_fee" label="文件费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={6}><Form.Item name="pss" label="PSS旺季附加费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="gri" label="GRI综合涨价"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="seal_fee" label="铅封费"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="other_fee" label="其他费用"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                    </Row>
                    <Form.Item name="remarks" label="备注">
                        <Input.TextArea rows={2} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 智能比价对话框 */}
            <Modal
                title={<Space><ThunderboltOutlined /> 智能比价</Space>}
                open={compareVisible}
                onCancel={() => { setCompareVisible(false); setCompareResults([]); }}
                footer={null}
                width={900}
            >
                <Form form={compareForm} layout="inline" style={{ marginBottom: 16 }}>
                    <Form.Item name="origin" label="起运港" rules={[{ required: true }]}>
                        <Input placeholder="CNSHA" style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item name="destination" label="目的港" rules={[{ required: true }]}>
                        <Input placeholder="USLAX" style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item name="container_type" label="柜型">
                        <Select placeholder="选择柜型" style={{ width: 120 }} allowClear>
                            <Option value="20GP">20GP</Option>
                            <Option value="40GP">40GP</Option>
                            <Option value="40HQ">40HQ</Option>
                        </Select>
                    </Form.Item>
                    <Form.Item>
                        <Button type="primary" onClick={handleCompare} loading={compareLoading}>开始比价</Button>
                    </Form.Item>
                </Form>

                {compareResults.length > 0 && (
                    <>
                        <Row gutter={16} style={{ marginBottom: 16 }}>
                            <Col span={8}>
                                <Statistic title="可选方案" value={compareResults.length} suffix="个" />
                            </Col>
                            <Col span={8}>
                                <Statistic title="最低价格" value={compareResults[0]?.total_fee || 0} prefix="$" />
                            </Col>
                            <Col span={8}>
                                <Statistic title="价格区间" value={`$${compareResults[0]?.total_fee || 0} - $${compareResults[compareResults.length - 1]?.total_fee || 0}`} />
                            </Col>
                        </Row>
                        <Table
                            columns={compareColumns}
                            dataSource={compareResults}
                            rowKey="id"
                            pagination={false}
                            size="small"
                        />
                    </>
                )}
            </Modal>
        </div>
    );
};

export default Rates;
