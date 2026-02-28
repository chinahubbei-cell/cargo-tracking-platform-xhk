import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { Input, Button, Card, Tag, Switch, Select, message, Spin, Alert } from 'antd';
import { SaveOutlined, LoadingOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import './NotificationSettings.css';

// 通知设置类型定义
interface NotificationSettings {
    // 预警通知
    alertCritical: boolean;
    alertWarning: boolean;
    alertInfo: boolean;
    // 运单通知
    shipmentStatus: boolean;
    shipmentDelay: boolean;
    // 设备通知
    deviceOffline: boolean;
    deviceLowBattery: boolean;
    deviceTempAnomaly: boolean;
    // 邮件通知
    emailEnabled: boolean;
    emailAddress: string;
    // 免打扰
    quietHoursEnabled: boolean;
    quietHoursStart: string;
    quietHoursEnd: string;
}

// 默认设置
const DEFAULT_SETTINGS: NotificationSettings = {
    alertCritical: true,
    alertWarning: true,
    alertInfo: false,
    shipmentStatus: true,
    shipmentDelay: true,
    deviceOffline: true,
    deviceLowBattery: true,
    deviceTempAnomaly: true,
    emailEnabled: false,
    emailAddress: '',
    quietHoursEnabled: false,
    quietHoursStart: '22:00',
    quietHoursEnd: '08:00',
};

// 时间选项 - 提取为常量避免重复创建
const TIME_OPTIONS = Array.from({ length: 24 }, (_, i) => {
    const time = `${i.toString().padStart(2, '0')}:00`;
    return { value: time, label: time };
});

// 通用的开关设置行组件
interface SettingRowProps {
    label: React.ReactNode;
    checked: boolean;
    onChange: (checked: boolean) => void;
    disabled?: boolean;
}

const SettingRow: React.FC<SettingRowProps> = ({ label, checked, onChange, disabled }) => (
    <div className="setting-row">
        <span className="setting-label">{label}</span>
        <Switch checked={checked} onChange={onChange} disabled={disabled} />
    </div>
);

// 预警通知卡片
interface AlertNotificationCardProps {
    settings: NotificationSettings;
    onChange: (key: keyof NotificationSettings, value: boolean) => void;
}

const AlertNotificationCard: React.FC<AlertNotificationCardProps> = ({ settings, onChange }) => (
    <Card size="small" title="预警通知" className="notification-card">
        <SettingRow
            label={<><Tag color="red">紧急</Tag> 紧急预警通知</>}
            checked={settings.alertCritical}
            onChange={(checked) => onChange('alertCritical', checked)}
        />
        <SettingRow
            label={<><Tag color="orange">警告</Tag> 警告级别通知</>}
            checked={settings.alertWarning}
            onChange={(checked) => onChange('alertWarning', checked)}
        />
        <SettingRow
            label={<><Tag color="blue">提示</Tag> 一般提示通知</>}
            checked={settings.alertInfo}
            onChange={(checked) => onChange('alertInfo', checked)}
        />
    </Card>
);

// 运单通知卡片
const ShipmentNotificationCard: React.FC<AlertNotificationCardProps> = ({ settings, onChange }) => (
    <Card size="small" title="运单通知" className="notification-card">
        <SettingRow
            label="运单状态变更通知（发货、到达、签收等）"
            checked={settings.shipmentStatus}
            onChange={(checked) => onChange('shipmentStatus', checked)}
        />
        <SettingRow
            label="运单延误预警通知"
            checked={settings.shipmentDelay}
            onChange={(checked) => onChange('shipmentDelay', checked)}
        />
    </Card>
);

// 设备通知卡片
const DeviceNotificationCard: React.FC<AlertNotificationCardProps> = ({ settings, onChange }) => (
    <Card size="small" title="设备通知" className="notification-card">
        <SettingRow
            label="设备离线通知"
            checked={settings.deviceOffline}
            onChange={(checked) => onChange('deviceOffline', checked)}
        />
        <SettingRow
            label="设备低电量预警（低于20%）"
            checked={settings.deviceLowBattery}
            onChange={(checked) => onChange('deviceLowBattery', checked)}
        />
        <SettingRow
            label="温度异常通知"
            checked={settings.deviceTempAnomaly}
            onChange={(checked) => onChange('deviceTempAnomaly', checked)}
        />
    </Card>
);

// 邮件通知卡片
interface EmailNotificationCardProps {
    settings: NotificationSettings;
    onChange: (key: keyof NotificationSettings, value: boolean | string) => void;
    emailError: string;
}

const EmailNotificationCard: React.FC<EmailNotificationCardProps> = ({ settings, onChange, emailError }) => (
    <Card size="small" title="邮件通知" className="notification-card">
        <SettingRow
            label="启用邮件通知"
            checked={settings.emailEnabled}
            onChange={(checked) => onChange('emailEnabled', checked)}
        />
        <div className="email-input-wrapper">
            <label className="input-label">通知邮箱</label>
            <Input
                placeholder="请输入接收通知的邮箱地址"
                value={settings.emailAddress}
                onChange={(e) => onChange('emailAddress', e.target.value)}
                disabled={!settings.emailEnabled}
                status={emailError ? 'error' : undefined}
            />
            {emailError && <div className="error-text">{emailError}</div>}
        </div>
    </Card>
);

// 免打扰时段卡片
interface QuietHoursCardProps {
    settings: NotificationSettings;
    onChange: (key: keyof NotificationSettings, value: boolean | string) => void;
    timeError: string;
}

const QuietHoursCard: React.FC<QuietHoursCardProps> = ({ settings, onChange, timeError }) => (
    <Card size="small" title="免打扰时段" className="notification-card">
        <SettingRow
            label="启用免打扰时段"
            checked={settings.quietHoursEnabled}
            onChange={(checked) => onChange('quietHoursEnabled', checked)}
        />
        <div className="time-range-wrapper">
            <div className="time-select">
                <label className="input-label">开始时间</label>
                <Select
                    value={settings.quietHoursStart}
                    onChange={(value) => onChange('quietHoursStart', value)}
                    disabled={!settings.quietHoursEnabled}
                    options={TIME_OPTIONS}
                    style={{ width: '100%' }}
                />
            </div>
            <div className="time-select">
                <label className="input-label">结束时间</label>
                <Select
                    value={settings.quietHoursEnd}
                    onChange={(value) => onChange('quietHoursEnd', value)}
                    disabled={!settings.quietHoursEnabled}
                    options={TIME_OPTIONS}
                    style={{ width: '100%' }}
                />
            </div>
        </div>
        {timeError && <div className="error-text">{timeError}</div>}
        {settings.quietHoursEnabled && !timeError && (
            <div className="quiet-hours-hint">
                在此时段内，除紧急预警外的通知将被静音
            </div>
        )}
    </Card>
);

// 主组件
const NotificationConfig: React.FC = () => {
    const { user } = useAuthStore();
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [settings, setSettings] = useState<NotificationSettings>({
        ...DEFAULT_SETTINGS,
        emailAddress: user?.email || '',
    });
    const [isDirty, setIsDirty] = useState(false);

    // 加载设置
    useEffect(() => {
        loadSettings();
    }, []);

    const loadSettings = useCallback(async () => {
        setLoading(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 300));
            // 从API加载后设置
            // setSettings(response.data);
        } catch (error) {
            message.error('加载通知设置失败');
        } finally {
            setLoading(false);
        }
    }, []);

    // 处理设置变更
    const handleChange = useCallback((key: keyof NotificationSettings, value: boolean | string) => {
        setSettings(prev => ({ ...prev, [key]: value }));
        setIsDirty(true);
    }, []);

    // 邮箱格式验证
    const emailError = useMemo(() => {
        if (!settings.emailEnabled) return '';
        if (!settings.emailAddress) return '请输入邮箱地址';
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(settings.emailAddress)) return '请输入有效的邮箱地址';
        return '';
    }, [settings.emailEnabled, settings.emailAddress]);

    // 免打扰时间验证
    const timeError = useMemo(() => {
        if (!settings.quietHoursEnabled) return '';
        const start = parseInt(settings.quietHoursStart.split(':')[0]);
        const end = parseInt(settings.quietHoursEnd.split(':')[0]);
        // 允许跨天设置（如22:00 - 08:00）
        if (start === end) return '开始时间和结束时间不能相同';
        return '';
    }, [settings.quietHoursEnabled, settings.quietHoursStart, settings.quietHoursEnd]);

    // 保存设置
    const handleSave = useCallback(async () => {
        // 验证
        if (emailError) {
            message.error(emailError);
            return;
        }
        if (timeError) {
            message.error(timeError);
            return;
        }

        setSaving(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 500));
            message.success('通知设置已保存');
            setIsDirty(false);
        } catch (error) {
            message.error('保存设置失败，请重试');
        } finally {
            setSaving(false);
        }
    }, [settings, emailError, timeError]);

    // 重置设置
    const handleReset = useCallback(() => {
        setSettings({
            ...DEFAULT_SETTINGS,
            emailAddress: user?.email || '',
        });
        setIsDirty(true);
    }, [user?.email]);

    if (loading) {
        return (
            <div className="notification-config">
                <h3 className="notification-title">通知设置</h3>
                <div className="loading-container">
                    <Spin indicator={<LoadingOutlined />} tip="加载设置中..." />
                </div>
            </div>
        );
    }

    return (
        <div className="notification-config">
            <h3 className="notification-title">通知设置</h3>

            <AlertNotificationCard settings={settings} onChange={handleChange} />
            <ShipmentNotificationCard settings={settings} onChange={handleChange} />
            <DeviceNotificationCard settings={settings} onChange={handleChange} />
            <EmailNotificationCard
                settings={settings}
                onChange={handleChange}
                emailError={emailError}
            />
            <QuietHoursCard
                settings={settings}
                onChange={handleChange}
                timeError={timeError}
            />

            <div className="action-buttons">
                <Button onClick={handleReset} disabled={saving}>
                    恢复默认
                </Button>
                <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={handleSave}
                    loading={saving}
                    disabled={!isDirty || !!emailError || !!timeError}
                >
                    保存设置
                </Button>
            </div>

            {isDirty && (
                <Alert
                    message="您有未保存的更改"
                    type="warning"
                    showIcon
                    className="unsaved-alert"
                />
            )}
        </div>
    );
};

export default NotificationConfig;
