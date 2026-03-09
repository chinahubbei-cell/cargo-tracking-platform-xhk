import React, { useEffect, useState, useCallback } from 'react';
import {
    Table, Button, Space, Tag, Modal, Form, Input, InputNumber,
    Select, message, Typography, Row, Col, Drawer, Descriptions,
    Tabs, Timeline, Popconfirm, Card, Divider, Radio, Badge
} from 'antd';
import {
    PlusOutlined, EyeOutlined, CheckOutlined, CloseOutlined,
    ReloadOutlined, FileTextOutlined, DollarOutlined,
    AuditOutlined, StopOutlined, SendOutlined, FileDoneOutlined
} from '@ant-design/icons';
import { orderApi, orgApi } from '../services/api';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { TextArea } = Input;

// ==================== 状态标签映射 ====================

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

const contractStatusMap: Record<string, { label: string; color: string }> = {
    not_generated: { label: '未生成', color: 'default' },
    generated: { label: '已生成', color: 'blue' },
    signing: { label: '签约中', color: 'cyan' },
    signed_online: { label: '在线签约', color: 'green' },
    signed_offline: { label: '线下签约', color: 'green' },
    invalid: { label: '已失效', color: 'red' },
};

const paymentStatusMap: Record<string, { label: string; color: string }> = {
    pending_payment: { label: '待支付', color: 'orange' },
    paid: { label: '已支付', color: 'green' },
};

const invoiceStatusMap: Record<string, { label: string; color: string }> = {
    not_requested: { label: '未申请', color: 'default' },
    requested: { label: '已申请', color: 'orange' },
    approved: { label: '审核通过', color: 'blue' },
    issued: { label: '已开票', color: 'green' },
    delivered: { label: '已送达', color: 'green' },
    rejected: { label: '已驳回', color: 'red' },
};

const orderTypeMap: Record<string, string> = {
    purchase: '新购',
    renewal: '续费',
};

const sourceMap: Record<string, string> = {
    admin: '后台',
    tracking: '追踪平台',
};

const StatusTag: React.FC<{ value: string; map: Record<string, { label: string; color: string }> }> = ({ value, map }) => {
    const info = map[value] || { label: value, color: 'default' };
    return <Tag color={info.color}>{info.label}</Tag>;
};

// ==================== 主组件 ====================

const Orders: React.FC = () => {
    const [data, setData] = useState<any[]>([]);
    const [orgs, setOrgs] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [pagination, setPagination] = useState({ page: 1, pageSize: 20, total: 0 });
    const [filters, setFilters] = useState<any>({});

    // 弹窗
    const [createVisible, setCreateVisible] = useState(false);
    const [detailVisible, setDetailVisible] = useState(false);
    const [reviewVisible, setReviewVisible] = useState(false);
    const [paymentVisible, setPaymentVisible] = useState(false);
    const [invoiceIssueVisible, setInvoiceIssueVisible] = useState(false);
    const [invoiceReviewVisible, setInvoiceReviewVisible] = useState(false);
    const [voidVisible, setVoidVisible] = useState(false);

    const [currentOrder, setCurrentOrder] = useState<any>(null);
    const [detailOrder, setDetailOrder] = useState<any>(null);

    const [createForm] = Form.useForm();
    const [reviewForm] = Form.useForm();
    const [paymentForm] = Form.useForm();
    const [invoiceIssueForm] = Form.useForm();
    const [invoiceReviewForm] = Form.useForm();
    const [voidForm] = Form.useForm();

    // ==================== 数据加载 ====================

    const fetchData = useCallback(async (page = 1) => {
        setLoading(true);
        try {
            const res = await orderApi.list({ ...filters, page, page_size: pagination.pageSize });
            if (res.data.success) {
                setData(res.data.data || []);
                setPagination(prev => ({
                    ...prev,
                    page,
                    total: res.data.pagination?.total || 0,
                }));
            }
        } finally {
            setLoading(false);
        }
    }, [filters, pagination.pageSize]);

    const fetchOrgs = async () => {
        try {
            const res = await orgApi.list({ page_size: 500 });
            if (res.data.success) {
                setOrgs(res.data.data || []);
            }
        } catch { /* ignore */ }
    };

    useEffect(() => { fetchData(); fetchOrgs(); }, []);
    useEffect(() => { fetchData(1); }, [filters]);

    const fetchDetail = async (id: string) => {
        try {
            const res = await orderApi.get(id);
            if (res.data.success) {
                setDetailOrder(res.data.data);
            }
        } catch {
            message.error('获取订单详情失败');
        }
    };

    // ==================== 操作处理 ====================

    const handleCreate = () => {
        createForm.resetFields();
        createForm.setFieldsValue({ order_type: 'purchase', service_years: 1, items: [{}] });
        setCreateVisible(true);
    };

    const handleCreateSubmit = async () => {
        try {
            const values = await createForm.validateFields();
            await orderApi.create(values);
            message.success('订单创建成功');
            setCreateVisible(false);
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return; // validation error
            message.error('创建失败');
        }
    };

    const handleSubmitReview = async (id: string) => {
        try {
            await orderApi.submitReview(id);
            message.success('已提交审核');
            fetchData();
            if (detailOrder?.id === id) fetchDetail(id);
        } catch { message.error('操作失败'); }
    };

    const handleReview = (record: any) => {
        setCurrentOrder(record);
        reviewForm.resetFields();
        setReviewVisible(true);
    };

    const handleReviewSubmit = async () => {
        try {
            const values = await reviewForm.validateFields();
            await orderApi.review(currentOrder.id, values);
            message.success(values.action === 'approve' ? '审核通过' : '已驳回');
            setReviewVisible(false);
            fetchData();
            if (detailOrder?.id === currentOrder.id) fetchDetail(currentOrder.id);
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('操作失败');
        }
    };

    const handleGenerateContract = async (id: string) => {
        try {
            await orderApi.generateContract(id);
            message.success('合同已生成');
            fetchData();
            if (detailOrder?.id === id) fetchDetail(id);
        } catch { message.error('操作失败'); }
    };

    const handleConfirmOfflineSign = async (id: string) => {
        try {
            await orderApi.confirmOfflineSign(id);
            message.success('签约已确认');
            fetchData();
            if (detailOrder?.id === id) fetchDetail(id);
        } catch { message.error('操作失败'); }
    };

    const handlePayment = (record: any) => {
        setCurrentOrder(record);
        paymentForm.resetFields();
        setPaymentVisible(true);
    };

    const handlePaymentSubmit = async () => {
        try {
            const values = await paymentForm.validateFields();
            await orderApi.confirmPayment(currentOrder.id, values);
            message.success('支付已确认');
            setPaymentVisible(false);
            fetchData();
            if (detailOrder?.id === currentOrder.id) fetchDetail(currentOrder.id);
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('操作失败');
        }
    };

    const handleVoid = (record: any) => {
        setCurrentOrder(record);
        voidForm.resetFields();
        setVoidVisible(true);
    };

    const handleVoidSubmit = async () => {
        try {
            const values = await voidForm.validateFields();
            await orderApi.void(currentOrder.id, values);
            message.success('订单已作废');
            setVoidVisible(false);
            fetchData();
            if (detailOrder?.id === currentOrder.id) fetchDetail(currentOrder.id);
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('操作失败');
        }
    };

    const handleInvoiceReview = (record: any) => {
        setCurrentOrder(record);
        invoiceReviewForm.resetFields();
        setInvoiceReviewVisible(true);
    };

    const handleInvoiceReviewSubmit = async () => {
        try {
            const values = await invoiceReviewForm.validateFields();
            await orderApi.invoiceReview(currentOrder.id, values);
            message.success(values.action === 'approve' ? '开票审核通过' : '开票已驳回');
            setInvoiceReviewVisible(false);
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('操作失败');
        }
    };

    const handleIssueInvoice = (record: any) => {
        setCurrentOrder(record);
        invoiceIssueForm.resetFields();
        setInvoiceIssueVisible(true);
    };

    const handleIssueInvoiceSubmit = async () => {
        try {
            const values = await invoiceIssueForm.validateFields();
            await orderApi.issueInvoice(currentOrder.id, values);
            message.success('开票完成');
            setInvoiceIssueVisible(false);
            fetchData();
        } catch (err: any) {
            if (err?.errorFields) return;
            message.error('操作失败');
        }
    };

    const handleFulfilling = async (id: string) => {
        try {
            await orderApi.startFulfilling(id);
            message.success('已开始履约');
            fetchData();
            if (detailOrder?.id === id) fetchDetail(id);
        } catch { message.error('操作失败'); }
    };

    const handleComplete = async (id: string) => {
        try {
            await orderApi.complete(id);
            message.success('订单已完成');
            fetchData();
            if (detailOrder?.id === id) fetchDetail(id);
        } catch { message.error('操作失败'); }
    };

    // ==================== 列定义 ====================

    const columns = [
        {
            title: '订单号', dataIndex: 'order_no', key: 'order_no', width: 170,
            render: (v: string, record: any) => (
                <a onClick={() => { fetchDetail(record.id); setDetailVisible(true); }}>{v}</a>
            ),
        },
        {
            title: '客户', dataIndex: 'org_name', key: 'org_name', width: 140,
            ellipsis: true,
        },
        {
            title: '类型', dataIndex: 'order_type', key: 'order_type', width: 70,
            render: (v: string) => <Tag color={v === 'purchase' ? 'blue' : 'cyan'}>{orderTypeMap[v] || v}</Tag>,
        },
        {
            title: '来源', dataIndex: 'source', key: 'source', width: 80,
            render: (v: string) => sourceMap[v] || v,
        },
        {
            title: '金额', dataIndex: 'amount_total', key: 'amount_total', width: 100,
            render: (v: number) => <Text strong>¥{v?.toFixed(2) || '0.00'}</Text>,
        },
        {
            title: '订单状态', dataIndex: 'order_status', key: 'order_status', width: 100,
            render: (v: string) => <StatusTag value={v} map={orderStatusMap} />,
        },
        {
            title: '合同', dataIndex: 'contract_status', key: 'contract_status', width: 90,
            render: (v: string) => <StatusTag value={v} map={contractStatusMap} />,
        },
        {
            title: '支付', dataIndex: 'payment_status', key: 'payment_status', width: 80,
            render: (v: string) => <StatusTag value={v} map={paymentStatusMap} />,
        },
        {
            title: '开票', dataIndex: 'invoice_status', key: 'invoice_status', width: 80,
            render: (v: string) => <StatusTag value={v} map={invoiceStatusMap} />,
        },
        {
            title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 150,
            render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作', key: 'action', width: 260, fixed: 'right' as const,
            render: (_: any, record: any) => {
                const s = record.order_status;
                const ps = record.payment_status;
                return (
                    <Space size="small" wrap>
                        <Button size="small" icon={<EyeOutlined />}
                            onClick={() => { fetchDetail(record.id); setDetailVisible(true); }}
                        />

                        {s === 'draft' && (
                            <Popconfirm title="确认提交审核？" onConfirm={() => handleSubmitReview(record.id)}>
                                <Button size="small" type="primary" icon={<SendOutlined />}>提审</Button>
                            </Popconfirm>
                        )}

                        {s === 'pending_review' && (
                            <Button size="small" type="primary" icon={<AuditOutlined />}
                                onClick={() => handleReview(record)}>审核</Button>
                        )}

                        {s === 'approved' && (
                            <Popconfirm title="确认生成合同？" onConfirm={() => handleGenerateContract(record.id)}>
                                <Button size="small" icon={<FileTextOutlined />}>生成合同</Button>
                            </Popconfirm>
                        )}

                        {s === 'contract_pending' && (
                            <Popconfirm title="确认线下签约完成？" onConfirm={() => handleConfirmOfflineSign(record.id)}>
                                <Button size="small" icon={<FileDoneOutlined />}>确认签约</Button>
                            </Popconfirm>
                        )}

                        {(s === 'payment_pending' || s === 'contract_signed') && (
                            <Button size="small" type="primary" icon={<DollarOutlined />}
                                onClick={() => handlePayment(record)}>确认到账</Button>
                        )}

                        {ps === 'paid' && record.invoice_status === 'requested' && (
                            <Button size="small" icon={<AuditOutlined />}
                                onClick={() => handleInvoiceReview(record)}>开票审核</Button>
                        )}

                        {record.invoice_status === 'approved' && (
                            <Button size="small" icon={<FileDoneOutlined />}
                                onClick={() => handleIssueInvoice(record)}>开票</Button>
                        )}

                        {ps === 'paid' && s === 'paid' && (
                            <Popconfirm title="确认开始履约？" onConfirm={() => handleFulfilling(record.id)}>
                                <Button size="small">履约</Button>
                            </Popconfirm>
                        )}

                        {s === 'fulfilling' && (
                            <Popconfirm title="确认订单完成？" onConfirm={() => handleComplete(record.id)}>
                                <Button size="small" type="primary">完成</Button>
                            </Popconfirm>
                        )}

                        {ps !== 'paid' && !['completed', 'cancelled'].includes(s) && (
                            <Button size="small" danger icon={<StopOutlined />}
                                onClick={() => handleVoid(record)}>作废</Button>
                        )}
                    </Space>
                );
            },
        },
    ];

    // ==================== 详情 Drawer ====================

    const renderDetail = () => {
        if (!detailOrder) return null;
        const o = detailOrder;

        const tabItems = [
            {
                key: 'basic',
                label: '基本信息',
                children: (
                    <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="订单号">{o.order_no}</Descriptions.Item>
                        <Descriptions.Item label="订单状态"><StatusTag value={o.order_status} map={orderStatusMap} /></Descriptions.Item>
                        <Descriptions.Item label="订单类型"><Tag color={o.order_type === 'purchase' ? 'blue' : 'cyan'}>{orderTypeMap[o.order_type]}</Tag></Descriptions.Item>
                        <Descriptions.Item label="来源">{sourceMap[o.source]}</Descriptions.Item>
                        <Descriptions.Item label="客户">{o.org_name}</Descriptions.Item>
                        <Descriptions.Item label="联系人">{o.contact_name} {o.phone}</Descriptions.Item>
                        <Descriptions.Item label="总金额"><Text strong style={{ color: '#f5222d', fontSize: 16 }}>¥{o.amount_total?.toFixed(2)}</Text></Descriptions.Item>
                        <Descriptions.Item label="有效期">{o.service_years || 1} 年</Descriptions.Item>
                        {o.shipping_address && <Descriptions.Item label="收货地址" span={2}>{o.shipping_address}</Descriptions.Item>}
                        {o.receiver_name && <Descriptions.Item label="收货人">{o.receiver_name} {o.receiver_phone}</Descriptions.Item>}
                        <Descriptions.Item label="备注" span={2}>{o.remark || '-'}</Descriptions.Item>
                        <Descriptions.Item label="创建时间">{dayjs(o.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                        <Descriptions.Item label="更新时间">{dayjs(o.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                        {o.reviewer_name && (
                            <>
                                <Descriptions.Item label="审核人">{o.reviewer_name}</Descriptions.Item>
                                <Descriptions.Item label="审核意见">{o.review_comment || '-'}</Descriptions.Item>
                            </>
                        )}
                    </Descriptions>
                ),
            },
            {
                key: 'items',
                label: `商品明细 (${o.items?.length || 0})`,
                children: (
                    <Table
                        dataSource={o.items || []}
                        rowKey="id"
                        size="small"
                        pagination={false}
                        columns={[
                            { title: '类型', dataIndex: 'item_type', render: (v: string) => v === 'device' ? '设备' : '续费服务' },
                            { title: '产品名称', dataIndex: 'product_name' },
                            { title: 'SKU/设备ID', render: (_: any, r: any) => r.sku_id || r.device_id || '-' },
                            { title: '数量', dataIndex: 'qty' },
                            { title: '设备单价', dataIndex: 'unit_price', render: (v: number) => `¥${v?.toFixed(2)}` },
                            { title: '服务费', dataIndex: 'service_price', render: (v: number) => v ? `¥${v?.toFixed(2)}/年` : '-' },
                        ]}
                    />
                ),
            },
            {
                key: 'contract',
                label: <Badge dot={o.contract_status !== 'not_generated'}>合同</Badge>,
                children: o.contract ? (
                    <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="合同编号">{o.contract.contract_no}</Descriptions.Item>
                        <Descriptions.Item label="合同状态"><StatusTag value={o.contract.contract_status} map={contractStatusMap} /></Descriptions.Item>
                        <Descriptions.Item label="签约方式">{o.contract.sign_mode === 'online' ? '在线签约' : o.contract.sign_mode === 'offline' ? '线下签约' : '待定'}</Descriptions.Item>
                        <Descriptions.Item label="签约时间">{o.contract.signed_at ? dayjs(o.contract.signed_at).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
                        {o.contract.company_name && <Descriptions.Item label="甲方公司" span={2}>{o.contract.company_name}</Descriptions.Item>}
                        {o.contract.file_url && <Descriptions.Item label="合同文件" span={2}><a href={o.contract.file_url} target="_blank" rel="noreferrer">查看文件</a></Descriptions.Item>}
                    </Descriptions>
                ) : <Text type="secondary">合同尚未生成</Text>,
            },
            {
                key: 'payment',
                label: <Badge dot={o.payment_status === 'paid'} color="green">支付</Badge>,
                children: o.payment ? (
                    <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="支付单号">{o.payment.payment_no}</Descriptions.Item>
                        <Descriptions.Item label="支付状态"><StatusTag value={o.payment.payment_status} map={paymentStatusMap} /></Descriptions.Item>
                        <Descriptions.Item label="支付渠道">{o.payment.channel === 'placeholder' ? '手工确认' : o.payment.channel}</Descriptions.Item>
                        <Descriptions.Item label="支付金额">¥{o.payment.amount?.toFixed(2)}</Descriptions.Item>
                        <Descriptions.Item label="支付时间">{o.payment.paid_at ? dayjs(o.payment.paid_at).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
                    </Descriptions>
                ) : <Text type="secondary">尚无支付记录</Text>,
            },
            {
                key: 'invoice',
                label: <Badge dot={['requested', 'approved'].includes(o.invoice_status)} color="orange">发票</Badge>,
                children: o.invoice ? (
                    <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="开票状态"><StatusTag value={o.invoice.invoice_status} map={invoiceStatusMap} /></Descriptions.Item>
                        <Descriptions.Item label="发票类型">{o.invoice.invoice_type === 'normal' ? '普通发票' : o.invoice.invoice_type}</Descriptions.Item>
                        <Descriptions.Item label="发票抬头">{o.invoice.title}</Descriptions.Item>
                        <Descriptions.Item label="税号">{o.invoice.tax_no}</Descriptions.Item>
                        <Descriptions.Item label="金额">¥{o.invoice.amount?.toFixed(2)}</Descriptions.Item>
                        <Descriptions.Item label="邮箱">{o.invoice.email || '-'}</Descriptions.Item>
                        {o.invoice.invoice_no && <Descriptions.Item label="发票号">{o.invoice.invoice_no}</Descriptions.Item>}
                        {o.invoice.issued_at && <Descriptions.Item label="开票时间">{dayjs(o.invoice.issued_at).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>}
                        {o.invoice.review_comment && <Descriptions.Item label="审核意见" span={2}>{o.invoice.review_comment}</Descriptions.Item>}
                    </Descriptions>
                ) : <Text type="secondary">未申请开票</Text>,
            },
            {
                key: 'logs',
                label: `操作日志 (${o.logs?.length || 0})`,
                children: (
                    <Timeline
                        items={(o.logs || []).map((log: any) => ({
                            color: log.action === 'reject' || log.action === 'void' ? 'red' : 'blue',
                            children: (
                                <div>
                                    <Text strong>{log.action}</Text>
                                    {log.from_status && <Text type="secondary"> {orderStatusMap[log.from_status]?.label || log.from_status} → {orderStatusMap[log.to_status]?.label || log.to_status}</Text>}
                                    <br />
                                    <Text type="secondary" style={{ fontSize: 12 }}>
                                        {log.operator_name || log.operator_id} · {dayjs(log.created_at).format('MM-DD HH:mm')}
                                    </Text>
                                    {log.note && <div style={{ marginTop: 4, color: '#666' }}>{log.note}</div>}
                                </div>
                            ),
                        }))}
                    />
                ),
            },
        ];

        return <Tabs items={tabItems} />;
    };

    // ==================== 渲染 ====================

    return (
        <div style={{ padding: 20 }}>
            {/* 筛选栏 */}
            <Card size="small" style={{ marginBottom: 16 }}>
                <Row gutter={16} align="middle">
                    <Col>
                        <Select
                            allowClear placeholder="订单类型" style={{ width: 120 }}
                            onChange={(v) => setFilters((p: any) => ({ ...p, order_type: v }))}
                        >
                            <Select.Option value="purchase">新购</Select.Option>
                            <Select.Option value="renewal">续费</Select.Option>
                        </Select>
                    </Col>
                    <Col>
                        <Select
                            allowClear placeholder="来源" style={{ width: 120 }}
                            onChange={(v) => setFilters((p: any) => ({ ...p, source: v }))}
                        >
                            <Select.Option value="admin">后台</Select.Option>
                            <Select.Option value="tracking">追踪平台</Select.Option>
                        </Select>
                    </Col>
                    <Col>
                        <Select
                            allowClear placeholder="订单状态" style={{ width: 130 }}
                            onChange={(v) => setFilters((p: any) => ({ ...p, order_status: v }))}
                        >
                            {Object.entries(orderStatusMap).map(([k, v]) => (
                                <Select.Option key={k} value={k}>{v.label}</Select.Option>
                            ))}
                        </Select>
                    </Col>
                    <Col>
                        <Input.Search
                            placeholder="搜索订单号/客户/联系人"
                            allowClear
                            style={{ width: 240 }}
                            onSearch={(v) => setFilters((p: any) => ({ ...p, keyword: v }))}
                        />
                    </Col>
                    <Col flex="auto" style={{ textAlign: 'right' }}>
                        <Space>
                            <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
                            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新增订单</Button>
                        </Space>
                    </Col>
                </Row>
            </Card>

            {/* 订单列表 */}
            <Table
                dataSource={data}
                columns={columns}
                rowKey="id"
                loading={loading}
                scroll={{ x: 1500 }}
                pagination={{
                    current: pagination.page,
                    pageSize: pagination.pageSize,
                    total: pagination.total,
                    showTotal: (t) => `共 ${t} 条`,
                    showSizeChanger: true,
                    onChange: (page, pageSize) => {
                        setPagination(prev => ({ ...prev, pageSize: pageSize || 20 }));
                        fetchData(page);
                    },
                }}
            />

            {/* ========== 创建订单弹窗 ========== */}
            <Modal
                title="新增订单"
                open={createVisible}
                onOk={handleCreateSubmit}
                onCancel={() => setCreateVisible(false)}
                width={650}
                okText="创建"
            >
                <Form form={createForm} layout="vertical">
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="org_id" label="客户" rules={[{ required: true, message: '请选择客户' }]}>
                                <Select showSearch optionFilterProp="children" placeholder="选择客户">
                                    {orgs.map((org: any) => (
                                        <Select.Option key={org.id} value={org.id}>{org.name}</Select.Option>
                                    ))}
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="order_type" label="订单类型" rules={[{ required: true }]}>
                                <Select>
                                    <Select.Option value="purchase">新购</Select.Option>
                                    <Select.Option value="renewal">续费</Select.Option>
                                </Select>
                            </Form.Item>
                        </Col>
                    </Row>

                    <Divider plain>商品明细</Divider>
                    <Form.List name="items">
                        {(fields, { add, remove }) => (
                            <>
                                {fields.map(({ key, name, ...restField }) => (
                                    <Card key={key} size="small" style={{ marginBottom: 8 }}
                                        extra={fields.length > 1 ? <Button size="small" danger onClick={() => remove(name)}>删除</Button> : null}>
                                        <Row gutter={12}>
                                            <Col span={8}>
                                                <Form.Item {...restField} name={[name, 'item_type']} label="类型"
                                                    rules={[{ required: true }]} initialValue="device">
                                                    <Select size="small">
                                                        <Select.Option value="device">设备</Select.Option>
                                                        <Select.Option value="service_renewal">续费服务</Select.Option>
                                                    </Select>
                                                </Form.Item>
                                            </Col>
                                            <Col span={16}>
                                                <Form.Item {...restField} name={[name, 'product_name']} label="产品名称"
                                                    rules={[{ required: true }]}>
                                                    <Input size="small" placeholder="如: X6 追货版" />
                                                </Form.Item>
                                            </Col>
                                        </Row>
                                        <Row gutter={12}>
                                            <Col span={6}>
                                                <Form.Item {...restField} name={[name, 'qty']} label="数量"
                                                    rules={[{ required: true }]} initialValue={1}>
                                                    <InputNumber size="small" min={1} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                            <Col span={6}>
                                                <Form.Item {...restField} name={[name, 'unit_price']} label="设备单价">
                                                    <InputNumber size="small" min={0} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                            <Col span={6}>
                                                <Form.Item {...restField} name={[name, 'service_price']} label="服务费/年">
                                                    <InputNumber size="small" min={0} style={{ width: '100%' }} />
                                                </Form.Item>
                                            </Col>
                                            <Col span={6}>
                                                <Form.Item {...restField} name={[name, 'sku_id']} label="SKU编码">
                                                    <Input size="small" placeholder="可选" />
                                                </Form.Item>
                                            </Col>
                                        </Row>
                                    </Card>
                                ))}
                                <Button type="dashed" onClick={() => add({ item_type: 'device', qty: 1 })} block icon={<PlusOutlined />}>
                                    添加商品
                                </Button>
                            </>
                        )}
                    </Form.List>

                    <Divider plain>其他信息</Divider>
                    <Row gutter={16}>
                        <Col span={8}>
                            <Form.Item name="service_years" label="有效期(年)">
                                <InputNumber min={1} max={10} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="contact_name" label="联系人">
                                <Input />
                            </Form.Item>
                        </Col>
                        <Col span={8}>
                            <Form.Item name="phone" label="联系电话">
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="shipping_address" label="收货地址">
                        <Input />
                    </Form.Item>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="receiver_name" label="收货人">
                                <Input />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="receiver_phone" label="收货人电话">
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="remark" label="备注">
                        <TextArea rows={2} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 审核弹窗 ========== */}
            <Modal
                title="订单审核"
                open={reviewVisible}
                onOk={handleReviewSubmit}
                onCancel={() => setReviewVisible(false)}
                width={480}
            >
                {currentOrder && (
                    <div style={{ marginBottom: 16 }}>
                        <Text type="secondary">订单号: </Text><Text strong>{currentOrder.order_no}</Text>
                        <br />
                        <Text type="secondary">客户: </Text><Text>{currentOrder.org_name}</Text>
                        <br />
                        <Text type="secondary">金额: </Text><Text strong style={{ color: '#f5222d' }}>¥{currentOrder.amount_total?.toFixed(2)}</Text>
                    </div>
                )}
                <Form form={reviewForm} layout="vertical">
                    <Form.Item name="action" label="审核结果" rules={[{ required: true, message: '请选择' }]}>
                        <Radio.Group>
                            <Radio.Button value="approve">
                                <CheckOutlined /> 通过
                            </Radio.Button>
                            <Radio.Button value="reject" style={{ marginLeft: 8 }}>
                                <CloseOutlined /> 驳回
                            </Radio.Button>
                        </Radio.Group>
                    </Form.Item>
                    <Form.Item name="comment" label="审核意见" rules={[{ required: true, message: '请填写审核意见' }]}>
                        <TextArea rows={3} placeholder="请填写审核意见（通过/驳回原因）" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 支付确认弹窗 ========== */}
            <Modal
                title="确认到账"
                open={paymentVisible}
                onOk={handlePaymentSubmit}
                onCancel={() => setPaymentVisible(false)}
                width={400}
            >
                {currentOrder && (
                    <div style={{ marginBottom: 16 }}>
                        <Text type="secondary">订单: </Text><Text strong>{currentOrder.order_no}</Text>
                        <br />
                        <Text type="secondary">应付金额: </Text><Text strong style={{ color: '#f5222d' }}>¥{currentOrder.amount_total?.toFixed(2)}</Text>
                    </div>
                )}
                <Form form={paymentForm} layout="vertical">
                    <Form.Item name="note" label="备注">
                        <TextArea rows={2} placeholder="支付备注（如: 银行转账流水号）" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 作废弹窗 ========== */}
            <Modal
                title="订单作废"
                open={voidVisible}
                onOk={handleVoidSubmit}
                onCancel={() => setVoidVisible(false)}
                width={400}
            >
                <Form form={voidForm} layout="vertical">
                    <Form.Item name="reason" label="作废原因" rules={[{ required: true, message: '请填写原因' }]}>
                        <TextArea rows={3} placeholder="请填写作废原因" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 开票审核弹窗 ========== */}
            <Modal
                title="开票审核"
                open={invoiceReviewVisible}
                onOk={handleInvoiceReviewSubmit}
                onCancel={() => setInvoiceReviewVisible(false)}
                width={450}
            >
                <Form form={invoiceReviewForm} layout="vertical">
                    <Form.Item name="action" label="审核结果" rules={[{ required: true }]}>
                        <Radio.Group>
                            <Radio.Button value="approve"><CheckOutlined /> 通过</Radio.Button>
                            <Radio.Button value="reject" style={{ marginLeft: 8 }}><CloseOutlined /> 驳回</Radio.Button>
                        </Radio.Group>
                    </Form.Item>
                    <Form.Item name="comment" label="审核意见">
                        <TextArea rows={2} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 开票弹窗 ========== */}
            <Modal
                title="开具发票"
                open={invoiceIssueVisible}
                onOk={handleIssueInvoiceSubmit}
                onCancel={() => setInvoiceIssueVisible(false)}
                width={450}
            >
                <Form form={invoiceIssueForm} layout="vertical">
                    <Form.Item name="invoice_no" label="发票号" rules={[{ required: true, message: '请输入发票号' }]}>
                        <Input placeholder="请输入发票号码" />
                    </Form.Item>
                    <Form.Item name="file_url" label="发票文件URL">
                        <Input placeholder="可选，发票PDF链接" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* ========== 订单详情 Drawer ========== */}
            <Drawer
                title={
                    <Space>
                        <span>订单详情</span>
                        {detailOrder && <Tag>{detailOrder.order_no}</Tag>}
                    </Space>
                }
                open={detailVisible}
                onClose={() => { setDetailVisible(false); setDetailOrder(null); }}
                width={700}
                extra={
                    detailOrder && (
                        <Space>
                            <StatusTag value={detailOrder.order_status} map={orderStatusMap} />
                        </Space>
                    )
                }
            >
                {renderDetail()}
            </Drawer>
        </div>
    );
};

export default Orders;
