import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { Select, Checkbox, Button, Tag, Divider, Space, message, Alert, Modal, Spin } from 'antd';
import { SaveOutlined, UndoOutlined, LoadingOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import './PermissionConfig.css';

// 权限定义类型
interface Permission {
    key: string;
    name: string;
    description: string;
    category: string;
    isCore?: boolean; // 核心权限，不可取消
}

// 角色类型
interface Role {
    key: string;
    name: string;
    label: string;
    color: string;
    isProtected?: boolean; // 受保护的角色，权限不可编辑
}

// 权限列表 - 提取为常量
const ALL_PERMISSIONS: Permission[] = [
    { key: 'dashboard_view', name: '仪表板查看', description: '查看系统仪表板数据', category: '基础功能', isCore: true },
    { key: 'shipment_view', name: '运单查看', description: '查看运单列表和详情', category: '运单管理' },
    { key: 'shipment_create', name: '创建运单', description: '创建新的运单', category: '运单管理' },
    { key: 'shipment_edit', name: '编辑运单', description: '编辑现有运单信息', category: '运单管理' },
    { key: 'shipment_delete', name: '删除运单', description: '删除运单记录', category: '运单管理' },
    { key: 'device_view', name: '设备查看', description: '查看设备列表和状态', category: '设备管理' },
    { key: 'device_bindshipment', name: '绑定设备', description: '将设备绑定到运单', category: '设备管理' },
    { key: 'device_track', name: '设备轨迹', description: '查看设备轨迹回放', category: '设备管理' },
    { key: 'alert_view', name: '预警查看', description: '查看系统预警信息', category: '预警中心' },
    { key: 'alert_handle', name: '预警处理', description: '处理和关闭预警', category: '预警中心' },
    { key: 'route_planning', name: '线路规划', description: '使用AI线路规划功能', category: '高级功能' },
    { key: 'user_manage', name: '用户管理', description: '管理系统用户', category: '系统管理' },
    { key: 'settings_manage', name: '系统设置', description: '修改系统配置', category: '系统管理' },
];

// 角色列表
const ROLES: Role[] = [
    { key: 'admin', name: '管理员', label: '超级管理员', color: 'red', isProtected: true },
    { key: 'operator', name: '操作员', label: '普通用户', color: 'blue' },
    { key: 'viewer', name: '查看者', label: '只读用户', color: 'green' },
];

// 默认权限配置
const DEFAULT_PERMISSIONS: Record<string, string[]> = {
    admin: ALL_PERMISSIONS.map(p => p.key), // 管理员拥有所有权限
    operator: ['dashboard_view', 'shipment_view', 'shipment_create', 'device_view', 'device_track', 'alert_view'],
    viewer: ['dashboard_view', 'shipment_view', 'device_view', 'alert_view'],
};

// 按类别分组权限 - 预计算
const PERMISSIONS_BY_CATEGORY = ALL_PERMISSIONS.reduce((acc, perm) => {
    if (!acc[perm.category]) acc[perm.category] = [];
    acc[perm.category].push(perm);
    return acc;
}, {} as Record<string, Permission[]>);

// 单个权限项组件
interface PermissionItemProps {
    permission: Permission;
    checked: boolean;
    disabled: boolean;
    onChange: (checked: boolean) => void;
}

const PermissionItem: React.FC<PermissionItemProps> = ({ permission, checked, disabled, onChange }) => (
    <div
        className={`permission-item ${checked ? 'checked' : ''} ${disabled ? 'disabled' : ''}`}
        onClick={() => !disabled && onChange(!checked)}
    >
        <Checkbox
            checked={checked}
            disabled={disabled}
            onChange={(e) => onChange(e.target.checked)}
        >
            <span className="permission-name">
                {permission.name}
                {permission.isCore && <Tag color="purple" className="core-tag">核心</Tag>}
            </span>
        </Checkbox>
        <div className="permission-description">{permission.description}</div>
    </div>
);

// 权限类别组件
interface PermissionCategoryProps {
    category: string;
    permissions: Permission[];
    selectedPermissions: string[];
    disabled: boolean;
    onPermissionChange: (permKey: string, checked: boolean) => void;
    onSelectAllCategory: (category: string, selected: boolean) => void;
}

const PermissionCategory: React.FC<PermissionCategoryProps> = ({
    category,
    permissions,
    selectedPermissions,
    disabled,
    onPermissionChange,
    onSelectAllCategory,
}) => {
    const allSelected = permissions.every(p => selectedPermissions.includes(p.key));
    const someSelected = permissions.some(p => selectedPermissions.includes(p.key)) && !allSelected;

    return (
        <div className="permission-category">
            <Divider className="category-divider">
                <div className="category-header">
                    <Checkbox
                        checked={allSelected}
                        indeterminate={someSelected}
                        disabled={disabled}
                        onChange={(e) => onSelectAllCategory(category, e.target.checked)}
                    >
                        <span className="category-title">{category}</span>
                    </Checkbox>
                    <span className="category-count">
                        {permissions.filter(p => selectedPermissions.includes(p.key)).length} / {permissions.length}
                    </span>
                </div>
            </Divider>
            <div className="permissions-grid">
                {permissions.map(perm => (
                    <PermissionItem
                        key={perm.key}
                        permission={perm}
                        checked={selectedPermissions.includes(perm.key)}
                        disabled={disabled || perm.isCore || false}
                        onChange={(checked) => onPermissionChange(perm.key, checked)}
                    />
                ))}
            </div>
        </div>
    );
};

// 主组件
const PermissionConfig: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [selectedRole, setSelectedRole] = useState<string>('operator');
    const [permissions, setPermissions] = useState<Record<string, string[]>>(DEFAULT_PERMISSIONS);
    const [originalPermissions, setOriginalPermissions] = useState<Record<string, string[]>>(DEFAULT_PERMISSIONS);
    const [isDirty, setIsDirty] = useState(false);

    // 当前角色信息
    const currentRole = useMemo(() => ROLES.find(r => r.key === selectedRole), [selectedRole]);
    const currentPermissions = useMemo(() => permissions[selectedRole] || [], [permissions, selectedRole]);
    const isRoleProtected = currentRole?.isProtected || false;

    // 加载权限配置
    useEffect(() => {
        loadPermissions();
    }, []);

    const loadPermissions = useCallback(async () => {
        setLoading(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 300));
            // API返回后设置
            // setPermissions(response.data);
            // setOriginalPermissions(response.data);
        } catch (error) {
            message.error('加载权限配置失败');
        } finally {
            setLoading(false);
        }
    }, []);

    // 检测是否有未保存的修改
    useEffect(() => {
        const hasChanges = JSON.stringify(permissions) !== JSON.stringify(originalPermissions);
        setIsDirty(hasChanges);
    }, [permissions, originalPermissions]);

    // 切换角色时检查未保存修改
    const handleRoleChange = useCallback((newRole: string) => {
        if (isDirty) {
            Modal.confirm({
                title: '未保存的修改',
                icon: <ExclamationCircleOutlined />,
                content: '切换角色将丢失当前未保存的修改，确定要继续吗？',
                okText: '确定切换',
                cancelText: '取消',
                onOk: () => {
                    setPermissions(originalPermissions);
                    setSelectedRole(newRole);
                },
            });
        } else {
            setSelectedRole(newRole);
        }
    }, [isDirty, originalPermissions]);

    // 处理单个权限变更
    const handlePermissionChange = useCallback((permKey: string, checked: boolean) => {
        // 核心权限不可取消
        const perm = ALL_PERMISSIONS.find(p => p.key === permKey);
        if (perm?.isCore && !checked) {
            message.warning('核心权限不可取消');
            return;
        }

        setPermissions(prev => {
            const current = prev[selectedRole] || [];
            if (checked) {
                return { ...prev, [selectedRole]: [...current, permKey] };
            } else {
                return { ...prev, [selectedRole]: current.filter(k => k !== permKey) };
            }
        });
    }, [selectedRole]);

    // 批量选择某类别所有权限
    const handleSelectAllCategory = useCallback((category: string, selected: boolean) => {
        const categoryPerms = PERMISSIONS_BY_CATEGORY[category] || [];

        setPermissions(prev => {
            const current = prev[selectedRole] || [];
            const categoryKeys = categoryPerms.map(p => p.key);
            const coreKeys = categoryPerms.filter(p => p.isCore).map(p => p.key);

            if (selected) {
                // 添加该类别所有权限
                const newPerms = [...new Set([...current, ...categoryKeys])];
                return { ...prev, [selectedRole]: newPerms };
            } else {
                // 移除非核心权限
                const newPerms = current.filter(k => !categoryKeys.includes(k) || coreKeys.includes(k));
                return { ...prev, [selectedRole]: newPerms };
            }
        });
    }, [selectedRole]);

    // 全选/取消全选
    const handleSelectAll = useCallback((selected: boolean) => {
        setPermissions(prev => {
            if (selected) {
                return { ...prev, [selectedRole]: ALL_PERMISSIONS.map(p => p.key) };
            } else {
                // 保留核心权限
                const corePerms = ALL_PERMISSIONS.filter(p => p.isCore).map(p => p.key);
                return { ...prev, [selectedRole]: corePerms };
            }
        });
    }, [selectedRole]);

    // 重置为默认权限
    const handleReset = useCallback(() => {
        Modal.confirm({
            title: '重置权限',
            icon: <ExclamationCircleOutlined />,
            content: `确定要将 ${currentRole?.name} 的权限重置为默认配置吗？`,
            okText: '确定重置',
            okType: 'danger',
            cancelText: '取消',
            onOk: () => {
                setPermissions(prev => ({
                    ...prev,
                    [selectedRole]: DEFAULT_PERMISSIONS[selectedRole] || [],
                }));
                message.success('已重置为默认权限');
            },
        });
    }, [selectedRole, currentRole]);

    // 保存权限
    const handleSave = useCallback(async () => {
        setSaving(true);
        try {
            // 模拟API调用 - 实际应替换为真实API
            await new Promise(resolve => setTimeout(resolve, 500));
            setOriginalPermissions({ ...permissions });
            message.success(`${currentRole?.name} 的权限已保存`);
        } catch (error) {
            message.error('保存权限失败，请重试');
        } finally {
            setSaving(false);
        }
    }, [permissions, currentRole]);

    // 撤销修改
    const handleUndo = useCallback(() => {
        setPermissions({ ...originalPermissions });
        message.info('已撤销修改');
    }, [originalPermissions]);

    if (loading) {
        return (
            <div className="permission-config">
                <h3 className="permission-title">权限管理</h3>
                <div className="loading-container">
                    <Spin indicator={<LoadingOutlined />} tip="加载权限配置..." />
                </div>
            </div>
        );
    }

    const allSelected = currentPermissions.length === ALL_PERMISSIONS.length;
    const someSelected = currentPermissions.length > 0 && !allSelected;

    return (
        <div className="permission-config">
            <h3 className="permission-title">权限管理</h3>

            {/* 角色选择 */}
            <div className="role-selector">
                <span className="role-label">选择角色：</span>
                <Select
                    value={selectedRole}
                    onChange={handleRoleChange}
                    style={{ width: 200 }}
                >
                    {ROLES.map(role => (
                        <Select.Option key={role.key} value={role.key}>
                            <span>
                                {role.name} ({role.key}) <Tag color={role.color}>{role.label}</Tag>
                            </span>
                        </Select.Option>
                    ))}
                </Select>

                {isRoleProtected && (
                    <Tag color="gold" className="protected-tag">受保护角色</Tag>
                )}
            </div>

            {/* 受保护角色提示 */}
            {isRoleProtected && (
                <Alert
                    message="超级管理员权限不可编辑"
                    description="管理员角色拥有所有权限且不可修改，这是为了确保系统安全。"
                    type="info"
                    showIcon
                    className="protected-alert"
                />
            )}

            {/* 批量操作 */}
            {!isRoleProtected && (
                <div className="batch-actions">
                    <Checkbox
                        checked={allSelected}
                        indeterminate={someSelected}
                        onChange={(e) => handleSelectAll(e.target.checked)}
                    >
                        全选所有权限
                    </Checkbox>
                    <Button size="small" onClick={handleReset}>
                        重置为默认
                    </Button>
                </div>
            )}

            {/* 权限列表按类别分组 */}
            {Object.entries(PERMISSIONS_BY_CATEGORY).map(([category, perms]) => (
                <PermissionCategory
                    key={category}
                    category={category}
                    permissions={perms}
                    selectedPermissions={currentPermissions}
                    disabled={isRoleProtected}
                    onPermissionChange={handlePermissionChange}
                    onSelectAllCategory={handleSelectAllCategory}
                />
            ))}

            {/* 操作按钮 */}
            <div className="action-bar">
                <Space>
                    <Button
                        icon={<UndoOutlined />}
                        onClick={handleUndo}
                        disabled={!isDirty || saving}
                    >
                        撤销修改
                    </Button>
                    <Button
                        type="primary"
                        icon={<SaveOutlined />}
                        loading={saving}
                        onClick={handleSave}
                        disabled={!isDirty || isRoleProtected}
                    >
                        保存权限设置
                    </Button>
                </Space>
                <span className="permission-count">
                    已开通 {currentPermissions.length} / {ALL_PERMISSIONS.length} 项权限
                </span>
            </div>

            {/* 未保存提醒 */}
            {isDirty && !isRoleProtected && (
                <Alert
                    message="您有未保存的权限修改"
                    type="warning"
                    showIcon
                    className="unsaved-alert"
                />
            )}
        </div>
    );
};

export default PermissionConfig;
