import React, { useEffect, useState } from 'react';
import { Modal, Form, InputNumber, Input, Radio, Typography, Descriptions, Tag, message } from 'antd';
import { orgApi } from '../services/api';
import dayjs from 'dayjs';

const { Text } = Typography;

interface RenewModalProps {
    visible: boolean;
    org: any;
    onClose: () => void;
    onSuccess: () => void;
}

const RenewModal: React.FC<RenewModalProps> = ({ visible, org, onClose, onSuccess }) => {
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [months, setMonths] = useState(1);

    useEffect(() => {
        if (visible) {
            form.resetFields();
            setMonths(1);
        }
    }, [visible, form]);

    const currentEnd = org?.service_end ? dayjs(org.service_end) : dayjs();
    const baseDate = currentEnd.isAfter(dayjs()) ? currentEnd : dayjs();
    const newEndDate = baseDate.add(months, 'month');

    const handleSubmit = async () => {
        const values = await form.validateFields();
        setLoading(true);
        try {
            await orgApi.renew(org.id, {
                period_months: months,
                amount: values.amount || 0,
                remark: values.remark || '',
            });
            message.success('续费成功');
            onSuccess();
        } catch {
            message.error('续费失败');
        } finally {
            setLoading(false);
        }
    };

    return (
        <Modal
            title="续费"
            open={visible}
            onOk={handleSubmit}
            onCancel={onClose}
            confirmLoading={loading}
            okText="确认续费"
            width={520}
        >
            <Descriptions column={1} style={{ marginBottom: 16 }} bordered size="small">
                <Descriptions.Item label="客户名称">{org?.name}</Descriptions.Item>
                <Descriptions.Item label="当前到期日期">
                    {org?.service_end ? dayjs(org.service_end).format('YYYY-MM-DD') : '未设置'}
                </Descriptions.Item>
            </Descriptions>

            <Form form={form} layout="vertical">
                <Form.Item label="续费时长" required>
                    <Radio.Group value={months} onChange={e => setMonths(e.target.value)}>
                        <Radio.Button value={1}>1个月</Radio.Button>
                        <Radio.Button value={3}>3个月</Radio.Button>
                        <Radio.Button value={6}>6个月</Radio.Button>
                        <Radio.Button value={12}>12个月</Radio.Button>
                    </Radio.Group>
                </Form.Item>

                <Form.Item label="续费金额（元）" name="amount">
                    <InputNumber min={0} precision={2} style={{ width: '100%' }} placeholder="选填" />
                </Form.Item>

                <Form.Item label="备注" name="remark">
                    <Input.TextArea rows={2} placeholder="选填" />
                </Form.Item>

                <div style={{ background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 6, padding: '12px 16px' }}>
                    <Text strong>续费后到期日期：</Text>
                    <Tag color="green" style={{ fontSize: 14, marginLeft: 8 }}>
                        {newEndDate.format('YYYY-MM-DD')}
                    </Tag>
                </div>
            </Form>
        </Modal>
    );
};

export default RenewModal;
