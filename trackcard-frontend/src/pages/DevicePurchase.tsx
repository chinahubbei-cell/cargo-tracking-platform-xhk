import React, { useEffect, useState, useCallback } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, InputNumber,
    Select, message, Typography, Row, Col, Drawer, Descriptions,
    Tabs, Timeline, Card, Divider, Alert
} from 'antd';
import {
    PlusOutlined, EyeOutlined, ReloadOutlined, EditOutlined,
    DollarOutlined, FileDoneOutlined
} from '@ant-design/icons';
import api from '../api/client';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { TextArea } = Input;

// 状态标签
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

const DevicePurchase: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [createVisible, setCreateVisible] = useState(false);
    const [detailVisible, setDetailVisible] = useState(false);
    const [signVisible, setSignVisible] = useState(false);
    const [invoiceVisible, setInvoiceVisible] = useState(false);
    const [detailOrder, setDetailOrder] = useState<any>(null);
    const [currentOrder, setCurrentOrder] = useState<any>(null);

    const [createForm] = Form.useForm();
    const [signForm] = Form.useForm();
    const [invoiceForm] = Form.useForm();

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const res = await api.get('/trade/orders', { params: { order_type: 'purchase' } });
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
            if (res.data.success) {
                setDetailOrder(res.data.data);
            }
        } catch { message.error('获取详情失败'); }
    };

    const handleCreate = async () => {
        try {
            const values = await createForm.validateFields();
            await api.post('/trade/orders/purchase', values);
            message.success('订单已提交，等待审核');
            setCreateVisible(false);
            createForm.resetFields();
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return;
            const msg = err.response?.data?.message || err.message || '未知错误';
            message.error('提交失败: ' + msg);
        }
    };

    const handleSignOnline = async () => {
        try {
            const values = await signForm.validateFields();
            await api.post(`/trade/orders/${currentOrder.id}/contract/sign-online`, values);
            message.success('签约成功');
            setSignVisible(false);
            fetchData();
            if (detailOrder?.id === currentOrder.id) fetchDetail(currentOrder.id);
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('签约失败');
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
                if (detailOrder?.id === id) fetchDetail(id);
            }
        } catch { message.error('操作失败'); }
    };

    const handleApplyInvoice = async () => {
        try {
            const values = await invoiceForm.validateFields();
            await api.post(`/trade/orders/${currentOrder.id}/invoice/apply`, values);
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
        { title: '订单状态', dataIndex: 'order_status', width: 100, render: (v: string) => <StatusTag value={v} /> },
        { title: '合同', dataIndex: 'contract_status', width: 90, render: (v: string) => <Tag>{v === 'not_generated' ? '未生成' : v === 'signed_online' ? '已签约' : v === 'signed_offline' ? '线下签约' : v}</Tag> },
        { title: '支付', dataIndex: 'payment_status', width: 80, render: (v: string) => <Tag color={v === 'paid' ? 'green' : 'orange'}>{v === 'paid' ? '已支付' : '待支付'}</Tag> },
        { title: '创建时间', dataIndex: 'created_at', width: 150, render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm') },
        {
            title: '操作', key: 'action', width: 200,
            render: (_: any, r: any) => (
                <Space size="small" wrap>
                    <Button size="small" icon={<EyeOutlined />} onClick={() => { fetchDetail(r.id); setDetailVisible(true); }} />
                    {(r.order_status === 'approved' || r.order_status === 'contract_pending') && (
                        <Button size="small" type="primary" icon={<EditOutlined />}
                            onClick={() => { setCurrentOrder(r); signForm.resetFields(); setSignVisible(true); }}>签约</Button>
                    )}
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
                    <Col><Title level={4} style={{ margin: 0 }}>设备购买</Title></Col>
                    <Col>
                        <Space>
                            <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
                            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
                                createForm.resetFields();
                                createForm.setFieldsValue({
                                    service_years: 1,
                                    items: [{
                                        product_name: 'X3 追货版',
                                        qty: 1,
                                        unit_price: 299,
                                        service_price: 100
                                    }]
                                });
                                setCreateVisible(true);
                            }}>新建购买订单</Button>
                        </Space>
                    </Col>
                </Row>
            </Card>

            <Table dataSource={data} columns={columns} rowKey="id" loading={loading} scroll={{ x: 900 }}
                pagination={{ showTotal: (t) => `共 ${t} 条`, showSizeChanger: true }} />

            {/* 创建订单 */}
            <Modal title="新建设备购买订单" open={createVisible} onOk={handleCreate}
                onCancel={() => setCreateVisible(false)} width={600} okText="提交">
                <Alert message="订单提交后将进入待审核状态，管理员审核通过后方可签约和支付。" type="info" showIcon style={{ marginBottom: 16 }} />
                <Form form={createForm} layout="vertical">
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="contact_name" label="联系人" rules={[{ required: true }]}>
                                <Input />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="phone" label="联系电话" rules={[{ required: true }]}>
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Divider plain>产品选择</Divider>
                    <Form.List name="items">
                        {(fields, { add, remove }) => (
                            <>
                                {fields.map(({ key, name, ...restField }) => (
                                    <Card key={key} size="small" style={{ marginBottom: 8 }}
                                        extra={fields.length > 1 ? <Button size="small" danger onClick={() => remove(name)}>删除</Button> : null}>
                                        <Row gutter={8}>
                                            <Col span={10}>
                                                <Form.Item {...restField} name={[name, 'product_name']} label="产品" rules={[{ required: true }]}>
                                                    <Select placeholder="选择产品">
                                                        <Select.Option value="X3 追货版">X3 追货版</Select.Option>
                                                        <Select.Option value="X6 追货版">X6 追货版</Select.Option>
                                                        <Select.Option value="X8 Pro 追货版">X8 Pro 追货版</Select.Option>
                                                    </Select>
                                                </Form.Item>
                                            </Col>
                                            <Col span={4}>
                                                <Form.Item {...restField} name={[name, 'qty']} label="数量" rules={[{ required: true }]}>
                                                    <InputNumber min={1} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                            <Col span={5}>
                                                <Form.Item {...restField} name={[name, 'unit_price']} label="设备单价">
                                                    <InputNumber min={0} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                            <Col span={5}>
                                                <Form.Item {...restField} name={[name, 'service_price']} label="服务费/年">
                                                    <InputNumber min={0} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                        </Row>
                                    </Card>
                                ))}
                                <Button type="dashed" onClick={() => add({ qty: 1 })} block icon={<PlusOutlined />}>添加产品</Button>
                            </>
                        )}
                    </Form.List>

                    <Divider plain>其他信息</Divider>
                    <Form.Item name="service_years" label="有效期(年)">
                        <InputNumber min={1} max={10} style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item name="shipping_address" label="收货地址">
                        <Input />
                    </Form.Item>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="receiver_name" label="收货人"><Input /></Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="receiver_phone" label="收货人电话"><Input /></Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="remark" label="备注"><TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>

            {/* 签约 */}
            <Modal title="在线签约" open={signVisible} onOk={handleSignOnline}
                onCancel={() => setSignVisible(false)} width={500} okText="确认签约">
                <Form form={signForm} layout="vertical">
                    <Form.Item name="company_name" label="甲方公司名称" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="credit_code" label="统一社会信用代码"><Input /></Form.Item>
                    <Row gutter={16}>
                        <Col span={12}><Form.Item name="legal_person" label="法人代表"><Input /></Form.Item></Col>
                        <Col span={12}><Form.Item name="contact_name" label="联系人"><Input /></Form.Item></Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={12}><Form.Item name="contact_phone" label="联系电话"><Input /></Form.Item></Col>
                        <Col span={12}><Form.Item name="contact_address" label="联系地址"><Input /></Form.Item></Col>
                    </Row>
                    <Divider plain>收货信息</Divider>
                    <Row gutter={16}>
                        <Col span={8}><Form.Item name="receiver_name" label="收货人"><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="receiver_phone" label="收货人电话"><Input /></Form.Item></Col>
                        <Col span={8}><Form.Item name="receiver_addr" label="收货地址"><Input /></Form.Item></Col>
                    </Row>
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
            <Drawer title={<Space>订单详情 {detailOrder && <Tag>{detailOrder.order_no}</Tag>}</Space>}
                open={detailVisible} onClose={() => { setDetailVisible(false); setDetailOrder(null); }}
                width={600}>
                {detailOrder && (
                    <Tabs items={[
                        {
                            key: 'info', label: '基本信息',
                            children: (
                                <Descriptions column={2} bordered size="small">
                                    <Descriptions.Item label="订单号">{detailOrder.order_no}</Descriptions.Item>
                                    <Descriptions.Item label="状态"><StatusTag value={detailOrder.order_status} /></Descriptions.Item>
                                    <Descriptions.Item label="总金额"><Text strong style={{ color: '#f5222d' }}>¥{detailOrder.amount_total?.toFixed(2)}</Text></Descriptions.Item>
                                    <Descriptions.Item label="有效期">{detailOrder.service_years || 1} 年</Descriptions.Item>
                                    <Descriptions.Item label="联系人">{detailOrder.contact_name}</Descriptions.Item>
                                    <Descriptions.Item label="电话">{detailOrder.phone}</Descriptions.Item>
                                    {detailOrder.shipping_address && <Descriptions.Item label="收货地址" span={2}>{detailOrder.shipping_address}</Descriptions.Item>}
                                    {detailOrder.review_comment && <Descriptions.Item label="审核意见" span={2}>{detailOrder.review_comment}</Descriptions.Item>}
                                </Descriptions>
                            ),
                        },
                        {
                            key: 'items', label: `产品 (${detailOrder.items?.length || 0})`,
                            children: (
                                <Table dataSource={detailOrder.items || []} rowKey="id" size="small" pagination={false}
                                    columns={[
                                        { title: '产品名称', dataIndex: 'product_name' },
                                        { title: '数量', dataIndex: 'qty' },
                                        { title: '设备单价', dataIndex: 'unit_price', render: (v: number) => `¥${v?.toFixed(2)}` },
                                        { title: '服务费/年', dataIndex: 'service_price', render: (v: number) => v ? `¥${v?.toFixed(2)}` : '-' },
                                    ]}
                                />
                            ),
                        },
                        {
                            key: 'logs', label: `日志 (${detailOrder.logs?.length || 0})`,
                            children: (
                                <Timeline
                                    items={(detailOrder.logs || []).map((log: any) => ({
                                        children: (
                                            <div>
                                                <Text strong>{log.action}</Text>
                                                <br />
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    {dayjs(log.created_at).format('MM-DD HH:mm')}
                                                </Text>
                                                {log.note && <div style={{ color: '#666' }}>{log.note}</div>}
                                            </div>
                                        ),
                                    }))}
                                />
                            ),
                        },
                    ]} />
                )}
            </Drawer>
        </div>
    );
};

export default DevicePurchase;
