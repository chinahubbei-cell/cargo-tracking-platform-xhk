import React, { useEffect, useState, useCallback } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, InputNumber,
    Select, message, Typography, Row, Col, Drawer, Descriptions,
    Tabs, Timeline, Card, Alert
} from 'antd';
import {
    PlusOutlined, EyeOutlined, ReloadOutlined,
    DollarOutlined, FileDoneOutlined
} from '@ant-design/icons';
import api from '../api/client';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { TextArea } = Input;

const orderStatusMap: Record<string, { label: string; color: string }> = {
    draft: { label: '草稿', color: 'default' },
    pending_review: { label: '待审核', color: 'orange' },
    rejected: { label: '已驳回', color: 'red' },
    approved: { label: '审核通过', color: 'blue' },
    contract_pending: { label: '待签约', color: 'cyan' },
    contract_signed: { label: '已签约', color: 'geekblue' },
    payment_pending: { label: '待支付', color: 'gold' },
    paid: { label: '已支付', color: 'lime' },
    invoice_pending: { label: '待开票', color: 'purple' },
    invoiced: { label: '已开票', color: 'magenta' },
    fulfilling: { label: '履约中', color: 'processing' },
    completed: { label: '已完成', color: 'green' },
    cancelled: { label: '已作废', color: '#bbb' },
};

const StatusTag: React.FC<{ value: string }> = ({ value }) => {
    const info = orderStatusMap[value] || { label: value, color: 'default' };
    return <Tag color={info.color}>{info.label}</Tag>;
};

const DeviceRenewal: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [createVisible, setCreateVisible] = useState(false);
    const [detailVisible, setDetailVisible] = useState(false);
    const [invoiceVisible, setInvoiceVisible] = useState(false);
    const [detailOrder, setDetailOrder] = useState<any>(null);
    const [currentOrder, setCurrentOrder] = useState<any>(null);

    const [createForm] = Form.useForm();
    const [invoiceForm] = Form.useForm();

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const res = await api.get('/trade/orders', { params: { order_type: 'renewal' } });
            if (res.data.success) {
                setData(res.data.data || []);
            }
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { fetchData(); }, []);

    const fetchDetail = async (id: string) => {
        try {
            const res = await api.get(`/trade/orders/${id}`);
            if (res.data.success) setDetailOrder(res.data.data);
        } catch { message.error('获取详情失败'); }
    };

    const handleCreate = async () => {
        try {
            const values = await createForm.validateFields();
            await api.post('/trade/orders/renewal', values);
            message.success('续费订单已提交，等待审核');
            setCreateVisible(false);
            createForm.resetFields();
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return;
            const msg = err.response?.data?.message || err.message || '未知错误';
            message.error('提交失败: ' + msg);
        }
    };

    const handleCreatePayment = async (id: string) => {
        try {
            const res = await api.post(`/trade/orders/${id}/payment/create`);
            if (res.data.success) {
                Modal.info({
                    title: '付款信息',
                    width: 500,
                    content: (
                        <Descriptions column={1} size="small" style={{ marginTop: 16 }}>
                            <Descriptions.Item label="应付金额"><Text strong style={{ color: '#f5222d' }}>¥{res.data.data.amount?.toFixed(2)}</Text></Descriptions.Item>
                            <Descriptions.Item label="收款户名">{res.data.data.account_name}</Descriptions.Item>
                            <Descriptions.Item label="开户银行">{res.data.data.bank_name}</Descriptions.Item>
                            <Descriptions.Item label="银行账号">{res.data.data.account_no}</Descriptions.Item>
                            <Descriptions.Item label="联行号">{res.data.data.bank_code}</Descriptions.Item>
                            <Descriptions.Item label="备注"><Text type="warning">{res.data.data.note}</Text></Descriptions.Item>
                        </Descriptions>
                    ),
                });
                fetchData();
            }
        } catch { message.error('操作失败'); }
    };

    const handleApplyInvoice = async () => {
        try {
            const values = await invoiceForm.validateFields();
            await api.post(`trade/orders/${currentOrder.id}/invoice/apply`, values);
            message.success('开票申请已提交');
            setInvoiceVisible(false);
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return;
            const msg = err.response?.data?.message || err.message || '未知错误';
            message.error('操作失败: ' + msg);
        }
    };

    const columns = [
        {
            title: '订单号', dataIndex: 'order_no', width: 170,
            render: (v: string, r: any) => <a onClick={() => { fetchDetail(r.id); setDetailVisible(true); }}>{v}</a>,
        },
        { title: '金额', dataIndex: 'amount_total', width: 100, render: (v: number) => <Text strong>¥{v?.toFixed(2)}</Text> },
        { title: '续费年数', dataIndex: 'service_years', width: 80, render: (v: number) => `${v || 1} 年` },
        { title: '状态', dataIndex: 'order_status', width: 100, render: (v: string) => <StatusTag value={v} /> },
        { title: '支付', dataIndex: 'payment_status', width: 80, render: (v: string) => <Tag color={v === 'paid' ? 'green' : 'orange'}>{v === 'paid' ? '已支付' : '待支付'}</Tag> },
        { title: '创建时间', dataIndex: 'created_at', width: 150, render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm') },
        {
            title: '操作', key: 'action', width: 180,
            render: (_: any, r: any) => (
                <Space size="small" wrap>
                    <Button size="small" icon={<EyeOutlined />} onClick={() => { fetchDetail(r.id); setDetailVisible(true); }} />
                    {(r.order_status === 'payment_pending' || r.order_status === 'contract_signed') && (
                        <Button size="small" icon={<DollarOutlined />} onClick={() => handleCreatePayment(r.id)}>支付</Button>
                    )}
                    {r.payment_status === 'paid' && r.invoice_status === 'not_requested' && (
                        <Button size="small" icon={<FileDoneOutlined />}
                            onClick={() => { setCurrentOrder(r); invoiceForm.resetFields(); setInvoiceVisible(true); }}>申请开票</Button>
                    )}
                </Space>
            ),
        },
    ];

    return (
        <div style={{ padding: 20 }}>
            <Card size="small" style={{ marginBottom: 16 }}>
                <Row justify="space-between" align="middle">
                    <Col><Title level={4} style={{ margin: 0 }}>设备续费</Title></Col>
                    <Col>
                        <Space>
                            <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
                                createForm.resetFields();
                                createForm.setFieldsValue({ service_years: 1 });
                                setCreateVisible(true);
                            }}>新建续费订单</Button>
                        </Space>
                    </Col>
                </Row>
            </Card>

            <Table dataSource={data} columns={columns} rowKey="id" loading={loading} scroll={{ x: 900 }}
                pagination={{ showTotal: (t) => `共 ${t} 条`, showSizeChanger: true }} />

            {/* 创建续费订单 */}
            <Modal title="新建设备续费订单" open={createVisible} onOk={handleCreate}
                onCancel={() => setCreateVisible(false)} width={550} okText="提交">
                <Alert message="续费订单提交后将进入待审核状态。续费成功后，设备服务期限将立即延长。" type="info" showIcon style={{ marginBottom: 16 }} />
                <Form form={createForm} layout="vertical">
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="contact_name" label="联系人" rules={[{ required: true }]}><Input /></Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="phone" label="联系电话" rules={[{ required: true }]}><Input /></Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="service_years" label="续费年数">
                        <InputNumber min={1} max={10} style={{ width: 120 }} />
                    </Form.Item>

                    <Form.List name="items">
                        {(fields, { add, remove }) => (
                            <>
                                {fields.map(({ key, name, ...restField }) => (
                                    <Card key={key} size="small" style={{ marginBottom: 8 }}
                                        extra={fields.length > 1 ? <Button size="small" danger onClick={() => remove(name)}>删除</Button> : null}>
                                        <Row gutter={8}>
                                            <Col span={12}>
                                                <Form.Item {...restField} name={[name, 'device_id']} label="设备ID" rules={[{ required: true }]}><Input placeholder="输入设备ID" /></Form.Item>
                                            </Col>
                                            <Col span={8}>
                                                <Form.Item {...restField} name={[name, 'product_name']} label="产品"><Input placeholder="如: X3" /></Form.Item>
                                            </Col>
                                            <Col span={4}>
                                                <Form.Item {...restField} name={[name, 'unit_price']} label="费用/年"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
                                            </Col>
                                        </Row>
                                    </Card>
                                ))}
                                <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>添加设备</Button>
                            </>
                        )}
                    </Form.List>
                    <Form.Item name="remark" label="备注" style={{ marginTop: 16 }}><TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>

            {/* 开票申请 */}
            <Modal title="申请开票" open={invoiceVisible} onOk={handleApplyInvoice}
                onCancel={() => setInvoiceVisible(false)} width={450} okText="提交">
                <Form form={invoiceForm} layout="vertical">
                    <Form.Item name="title" label="发票抬头" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="tax_no" label="税号" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="email" label="接收邮箱"><Input /></Form.Item>
                    <Form.Item name="address" label="邮寄地址"><Input /></Form.Item>
                </Form>
            </Modal>

            {/* 详情 */}
            <Drawer title={<Space>续费订单详情 {detailOrder && <Tag>{detailOrder.order_no}</Tag>}</Space>}
                open={detailVisible} onClose={() => { setDetailVisible(false); setDetailOrder(null); }}
                width={550}>
                {detailOrder && (
                    <Tabs items={[
                        {
                            key: 'info', label: '基本信息',
                            children: (
                                <Descriptions column={2} bordered size="small">
                                    <Descriptions.Item label="订单号">{detailOrder.order_no}</Descriptions.Item>
                                    <Descriptions.Item label="状态"><StatusTag value={detailOrder.order_status} /></Descriptions.Item>
                                    <Descriptions.Item label="总金额"><Text strong style={{ color: '#f5222d' }}>¥{detailOrder.amount_total?.toFixed(2)}</Text></Descriptions.Item>
                                    <Descriptions.Item label="续费年数">{detailOrder.service_years || 1} 年</Descriptions.Item>
                                    <Descriptions.Item label="联系人">{detailOrder.contact_name}</Descriptions.Item>
                                    <Descriptions.Item label="电话">{detailOrder.phone}</Descriptions.Item>
                                    {detailOrder.review_comment && <Descriptions.Item label="审核意见" span={2}>{detailOrder.review_comment}</Descriptions.Item>}
                                </Descriptions>
                            ),
                        },
                        {
                            key: 'items', label: `设备 (${detailOrder.items?.length || 0})`,
                            children: (
                                <Table dataSource={detailOrder.items || []} rowKey="id" size="small" pagination={false}
                                    columns={[
                                        { title: '设备ID', dataIndex: 'device_id' },
                                        { title: '产品', dataIndex: 'product_name' },
                                        { title: '续费费用/年', dataIndex: 'service_price', render: (v: number) => `¥${v?.toFixed(2)}` },
                                    ]}
                                />
                            ),
                        },
                        {
                            key: 'logs', label: `日志`,
                            children: (
                                <Timeline items={(detailOrder.logs || []).map((log: any) => ({
                                    children: (
                                        <div>
                                            <Text strong>{log.action}</Text>
                                            <br />
                                            <Text type="secondary" style={{ fontSize: 12 }}>{dayjs(log.created_at).format('MM-DD HH:mm')}</Text>
                                            {log.note && <div style={{ color: '#666' }}>{log.note}</div>}
                                        </div>
                                    ),
                                }))} />
                            ),
                        },
                    ]} />
                )}
            </Drawer>
        </div>
    );
};

export default DeviceRenewal;
