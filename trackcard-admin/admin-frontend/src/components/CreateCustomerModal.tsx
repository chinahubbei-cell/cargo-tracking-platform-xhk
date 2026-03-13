import React, { useState } from 'react';
import {
    Modal, Steps, Form, Input, InputNumber, Select, DatePicker, Radio,
    Row, Col, Descriptions, Tag, Button, message, Space
} from 'antd';
import { orgApi } from '../services/api';
import dayjs from 'dayjs';

interface CreateCustomerModalProps {
    visible: boolean;
    parentOptions: any[];
    onClose: () => void;
    onSuccess: () => void;
}

const CreateCustomerModal: React.FC<CreateCustomerModalProps> = ({ visible, parentOptions, onClose, onSuccess }) => {
    const [current, setCurrent] = useState(0);
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [durationMode, setDurationMode] = useState<number>(1);
    const [formData, setFormData] = useState<any>({});

    const isSubAccount = !!Form.useWatch('parent_id', form);

    const reset = () => {
        setCurrent(0);
        form.resetFields();
        setFormData({});
        setDurationMode(1);
    };

    const handleClose = () => {
        reset();
        onClose();
    };

    const steps = isSubAccount
        ? ['基本信息', '确认创建']
        : ['基本信息', '服务套餐', '确认创建'];

    const handleNext = async () => {
        try {
            if (current === 0) {
                const values = await form.validateFields([
                    'name', 'parent_id', 'company_name', 'credit_code', 'short_name',
                    'contact_name', 'contact_phone', 'contact_email', 'address', 'remark'
                ]);
                setFormData((prev: any) => ({ ...prev, ...values }));
            }
            if (current === 1 && !isSubAccount) {
                const values = await form.validateFields([
                    'service_status', 'service_start', 'max_devices', 'max_users', 'max_shipments'
                ]);
                setFormData((prev: any) => ({ ...prev, ...values }));
            }
            setCurrent(current + 1);
        } catch {
            // validation failed
        }
    };

    const handleSubmit = async () => {
        setLoading(true);
        try {
            const allValues = form.getFieldsValue(true);
            const payload: any = {
                name: allValues.name,
                parent_id: allValues.parent_id || '',
                short_name: allValues.short_name || '',
                company_name: allValues.company_name || '',
                credit_code: allValues.credit_code || '',
                contact_name: allValues.contact_name || '',
                contact_phone: allValues.contact_phone || '',
                contact_email: allValues.contact_email || '',
                address: allValues.address || '',
                remark: allValues.remark || '',
            };

            if (!allValues.parent_id) {
                payload.service_status = allValues.service_status || 'trial';
                payload.max_devices = allValues.max_devices || 10;
                payload.max_users = allValues.max_users || 5;
                payload.max_shipments = allValues.max_shipments || 100;
                if (allValues.service_start) {
                    payload.service_start = allValues.service_start.toISOString();
                }
                if (allValues.service_end) {
                    payload.service_end = allValues.service_end.toISOString();
                } else if (allValues.service_start && durationMode) {
                    const endDate = allValues.service_start.add(durationMode, 'month');
                    payload.service_end = endDate.toISOString();
                }
            }

            await orgApi.create(payload);
            message.success('客户创建成功');
            reset();
            onSuccess();
        } catch {
            message.error('创建失败');
        } finally {
            setLoading(false);
        }
    };

    const serviceStart = Form.useWatch('service_start', form);
    const computedEndDate = serviceStart ? dayjs(serviceStart).add(durationMode, 'month') : null;

    const renderStep0 = () => (
        <>
            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item name="name" label="客户名称" rules={[{ required: true, message: '请输入客户名称' }]}>
                        <Input placeholder="如：ABC物流有限公司" />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="parent_id" label="上级账号">
                        <Select allowClear placeholder="无（作为主账号）">
                            {parentOptions.map((org: any) => (
                                <Select.Option key={org.id} value={org.id}>{org.name}</Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                </Col>
            </Row>
            {!isSubAccount && (
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="company_name" label="公司名称" rules={[{ required: !isSubAccount, message: '请输入公司名称' }]}>
                            <Input placeholder="正式注册公司名称" />
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <Form.Item name="credit_code" label="社会信用代码">
                            <Input placeholder="18位统一社会信用代码" maxLength={18} />
                        </Form.Item>
                    </Col>
                </Row>
            )}
            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item name="short_name" label="简称">
                        <Input />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="contact_name" label="联系人">
                        <Input />
                    </Form.Item>
                </Col>
            </Row>
            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item name="contact_phone" label="联系电话">
                        <Input />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="contact_email" label="邮箱">
                        <Input />
                    </Form.Item>
                </Col>
            </Row>
            <Form.Item name="address" label="地址">
                <Input.TextArea rows={2} />
            </Form.Item>
            <Form.Item name="remark" label="备注">
                <Input.TextArea rows={2} />
            </Form.Item>
        </>
    );

    const renderStep1Service = () => (
        <>
            <Row gutter={16}>
                <Col span={8}>
                    <Form.Item name="service_status" label="服务状态" initialValue="trial">
                        <Select>
                            <Select.Option value="trial">试用</Select.Option>
                            <Select.Option value="active">正常</Select.Option>
                        </Select>
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="service_start" label="服务开始日期" initialValue={dayjs()}>
                        <DatePicker style={{ width: '100%' }} />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item label="服务时长">
                        <Radio.Group value={durationMode} onChange={e => setDurationMode(e.target.value)}>
                            <Radio.Button value={1}>1月</Radio.Button>
                            <Radio.Button value={3}>3月</Radio.Button>
                            <Radio.Button value={6}>6月</Radio.Button>
                            <Radio.Button value={12}>12月</Radio.Button>
                        </Radio.Group>
                    </Form.Item>
                </Col>
            </Row>
            {computedEndDate && (
                <div style={{ background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 6, padding: '8px 16px', marginBottom: 16 }}>
                    服务到期日期：<Tag color="green">{computedEndDate.format('YYYY-MM-DD')}</Tag>
                </div>
            )}
            <Row gutter={16}>
                <Col span={8}>
                    <Form.Item name="max_devices" label="最大设备数" initialValue={10}>
                        <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="max_users" label="最大用户数" initialValue={5}>
                        <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="max_shipments" label="月运单配额" initialValue={100}>
                        <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                </Col>
            </Row>
        </>
    );

    const renderConfirm = () => {
        const allValues = form.getFieldsValue(true);
        const parentName = parentOptions.find((o: any) => o.id === allValues.parent_id)?.name;
        return (
            <Descriptions bordered column={2} size="small">
                <Descriptions.Item label="客户名称">{allValues.name}</Descriptions.Item>
                <Descriptions.Item label="账号类型">
                    {allValues.parent_id ? <Tag color="cyan">二级账号</Tag> : <Tag color="blue">主账号</Tag>}
                </Descriptions.Item>
                {!allValues.parent_id && (
                    <>
                        <Descriptions.Item label="公司名称">{allValues.company_name || '-'}</Descriptions.Item>
                        <Descriptions.Item label="社会信用代码">{allValues.credit_code || '-'}</Descriptions.Item>
                    </>
                )}
                {allValues.parent_id && (
                    <Descriptions.Item label="上级账号" span={2}>{parentName || '-'}</Descriptions.Item>
                )}
                <Descriptions.Item label="简称">{allValues.short_name || '-'}</Descriptions.Item>
                <Descriptions.Item label="联系人">{allValues.contact_name || '-'}</Descriptions.Item>
                <Descriptions.Item label="联系电话">{allValues.contact_phone || '-'}</Descriptions.Item>
                <Descriptions.Item label="邮箱">{allValues.contact_email || '-'}</Descriptions.Item>
                <Descriptions.Item label="地址" span={2}>{allValues.address || '-'}</Descriptions.Item>
                {!allValues.parent_id && (
                    <>
                        <Descriptions.Item label="服务状态">
                            <Tag color={allValues.service_status === 'active' ? 'green' : 'blue'}>
                                {allValues.service_status === 'active' ? '正常' : '试用'}
                            </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="服务期限">
                            {allValues.service_start?.format?.('YYYY-MM-DD') || '-'} ~ {computedEndDate?.format('YYYY-MM-DD') || '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label="最大设备数">{allValues.max_devices || 10}</Descriptions.Item>
                        <Descriptions.Item label="最大用户数">{allValues.max_users || 5}</Descriptions.Item>
                        <Descriptions.Item label="月运单配额">{allValues.max_shipments || 100}</Descriptions.Item>
                    </>
                )}
            </Descriptions>
        );
    };

    const confirmStepIndex = isSubAccount ? 1 : 2;

    const renderStepContent = () => {
        if (current === 0) return renderStep0();
        if (current === 1 && !isSubAccount) return renderStep1Service();
        return renderConfirm();
    };

    return (
        <Modal
            title="新增客户"
            open={visible}
            onCancel={handleClose}
            width={700}
            footer={
                <Space>
                    {current > 0 && (
                        <Button onClick={() => setCurrent(current - 1)}>上一步</Button>
                    )}
                    {current < confirmStepIndex ? (
                        <Button type="primary" onClick={handleNext}>下一步</Button>
                    ) : (
                        <Button type="primary" loading={loading} onClick={handleSubmit}>确认创建</Button>
                    )}
                </Space>
            }
        >
            <Steps
                current={current}
                items={steps.map(title => ({ title }))}
                style={{ marginBottom: 24 }}
                size="small"
            />
            <Form form={form} layout="vertical">
                {renderStepContent()}
            </Form>
        </Modal>
    );
};

export default CreateCustomerModal;
