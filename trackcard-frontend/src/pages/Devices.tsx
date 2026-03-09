import React, { useEffect, useState, useCallback, lazy, Suspense } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { Table, Tag, Button, Space, Input, Modal, Form, message, Select, DatePicker, Tabs, Empty, Slider, Spin, Dropdown, Radio, Card } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined, DeleteOutlined, EditOutlined, UnorderedListOutlined, GlobalOutlined, PlayCircleOutlined, PauseOutlined, SettingOutlined, EyeOutlined, HistoryOutlined, EnvironmentOutlined, DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import * as XLSX from 'xlsx';
import { downloadExcel } from '../utils/downloadUtils';
import api from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { OrganizationTreeNode } from '../types';

type ViewMode = 'list' | 'map' | 'track';

// 供应商配置
const PROVIDERS: Record<string, { label: string; color: string }> = {
    kuaihuoyun: { label: '快货运', color: 'blue' },
    sinoiov: { label: '中交兴路', color: 'orange' },
};

interface DeviceData {
    id: string;
    name: string;
    type: string;
    provider: string;
    status: string;
    battery: number;
    latitude: number;
    longitude: number;
    external_device_id: string;
    speed: number;
    direction: number;
    temperature: number;
    humidity: number;
    last_update: string;
    created_at: string;
    // 绑定运单信息
    binding_status?: string;
    bound_transport_type?: string;
    bound_cargo_name?: string;
    bound_shipment_id?: string;
    locate_type?: number;
    service_end_at?: string;
    org_name?: string;
}

interface TrackPoint {
    latitude: number;
    longitude: number;
    speed: number;
    timestamp: string;
    temperature: number;
}


// 延迟加载地图组件
const DeviceMap = lazy(() => import('../components/DeviceMap'));
const TrackMap = lazy(() => import('../components/TrackMap'));

const Devices: React.FC = () => {
    const [searchParams] = useSearchParams();
    const [loading, setLoading] = useState(false);
    const [syncing, setSyncing] = useState(false); // 后台同步状态（不阻塞界面）
    const [devices, setDevices] = useState<DeviceData[]>([]);
    const [search, setSearch] = useState(searchParams.get('search') || '');
    const [modalVisible, setModalVisible] = useState(false);

    // 监听URL search参数变化
    useEffect(() => {
        const urlSearch = searchParams.get('search');
        if (urlSearch && urlSearch !== search) {
            setSearch(urlSearch);
        }
    }, [searchParams]);
    const [editingDevice, setEditingDevice] = useState<DeviceData | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('list');
    const [form] = Form.useForm();
    const { user } = useAuthStore();
    const navigate = useNavigate();

    // 轨迹回放状态
    const [trackDeviceId, setTrackDeviceId] = useState('');
    const [trackData, setTrackData] = useState<TrackPoint[]>([]);
    const [isPlaying, setIsPlaying] = useState(false);
    const [currentTrackIndex, setCurrentTrackIndex] = useState(0);
    const [playSpeed, setPlaySpeed] = useState(1);
    const [focusDevice, setFocusDevice] = useState<DeviceData | null>(null);
    // 时间范围选择状态
    const [trackTimeRange, setTrackTimeRange] = useState<'today' | 'yesterday' | 'last3days' | 'custom'>('today');
    const [customTimeRange, setCustomTimeRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
    const [customPickerVisible, setCustomPickerVisible] = useState(false);
    // 筛选状态
    const [filterTransportType, setFilterTransportType] = useState<string>('');
    const [filterBindingStatus, setFilterBindingStatus] = useState<string>('');
    const [filterOrgId, setFilterOrgId] = useState<string>('');
    const [organizations, setOrganizations] = useState<OrganizationTreeNode[]>([]);

    const canEdit = user?.role === 'admin' || user?.role === 'operator';

    // 展平组织树结构，用于筛选下拉框
    const flattenOrgs = (orgs: OrganizationTreeNode[], level = 0): { id: string; name: string; level: number }[] => {
        const result: { id: string; name: string; level: number }[] = [];
        for (const org of orgs) {
            result.push({ id: org.id, name: org.name, level });
            if (org.children && org.children.length > 0) {
                result.push(...flattenOrgs(org.children, level + 1));
            }
        }
        return result;
    };
    const flatOrganizations = flattenOrgs(organizations);


    // 加载设备列表（不触发外部同步）
    const loadDevices = useCallback(async (showLoading = true) => {
        if (showLoading) setLoading(true);
        try {
            const [devicesRes, orgsRes] = await Promise.all([
                api.getDevices({ search, syncExternal: false, org_id: filterOrgId }),
                api.getOrganizations({ tree: true }),
            ]);
            const devicesData = devicesRes.data || [];
            const normalizedDevices = devicesData.map((d: any) => ({
                ...d,
                provider: d.provider || 'kuaihuoyun',
                battery: d.battery || 0,
                speed: d.speed || 0,
                temperature: d.temperature || 0,
                humidity: d.humidity || 0,
            }));
            setDevices(normalizedDevices);
            if (orgsRes) setOrganizations(orgsRes);
        } catch {
            setDevices([]);
        } finally {
            if (showLoading) setLoading(false);
        }
    }, [search, filterOrgId]);

    // 后台静默同步（不阻塞界面）
    const syncDevicesInBackground = useCallback(async () => {
        if (syncing) return; // 避免重复同步
        setSyncing(true);
        message.loading({ content: '正在同步设备数据...', key: 'sync', duration: 0 });
        try {
            const res = await api.getDevices({ search, syncExternal: true });
            const devicesData = res.data || [];
            const normalizedDevices = devicesData.map((d: any) => ({
                ...d,
                provider: d.provider || 'kuaihuoyun',
                battery: d.battery || 0,
                speed: d.speed || 0,
                temperature: d.temperature || 0,
                humidity: d.humidity || 0,
            }));
            setDevices(normalizedDevices);
            message.success({ content: '同步完成', key: 'sync', duration: 2 });
        } catch {
            message.error({ content: '同步失败', key: 'sync', duration: 2 });
        } finally {
            setSyncing(false);
        }
    }, [search, syncing]);

    const isFirstRender = React.useRef(true);

    useEffect(() => {
        loadDevices();
    }, []);

    // 组织筛选变化时重新加载数据
    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        loadDevices();
    }, [filterOrgId]);

    // 轨迹回放动画
    useEffect(() => {
        if (isPlaying && trackData.length > 0) {
            const interval = setInterval(() => {
                setCurrentTrackIndex(prev => {
                    if (prev >= trackData.length - 1) {
                        setIsPlaying(false);
                        return prev;
                    }
                    return prev + 1;
                });
            }, 1000 / playSpeed);
            return () => clearInterval(interval);
        }
    }, [isPlaying, trackData.length, playSpeed]);

    const handleCreate = () => {
        setEditingDevice(null);
        form.resetFields();
        setModalVisible(true);
    };

    const handleEdit = (device: DeviceData) => {
        setEditingDevice(device);
        form.setFieldsValue(device);
        setModalVisible(true);
    };

    const handleDelete = async (id: string) => {
        try {
            await api.deleteDevice(id);
            message.success('删除成功');
            loadDevices();
        } catch {
            message.error('删除失败');
        }
    };

    const handleSubmit = async (values: any) => {
        try {
            if (editingDevice) {
                await api.updateDevice(editingDevice.id, values);
                message.success('更新成功');
            } else {
                await api.createDevice(values);
                message.success('创建成功');
            }
            setModalVisible(false);
            loadDevices();
        } catch (error: any) {
            message.error(error.response?.data?.error || '操作失败');
        }
    };

    const handleLoadTrack = async () => {
        if (!trackDeviceId) {
            message.warning('请输入设备号');
            return;
        }

        setLoading(true);
        try {
            // 查找设备的内部ID（如 XHK-001）基于 external_device_id
            const device = devices.find(d =>
                d.external_device_id === trackDeviceId || d.id === trackDeviceId
            );
            const deviceId = device?.id || trackDeviceId;

            // 根据用户选择的时间区间计算时间范围
            let startTime: string;
            let endTime: string;
            const now = dayjs();

            switch (trackTimeRange) {
                case 'today':
                    startTime = now.startOf('day').format('YYYY-MM-DD HH:mm:ss');
                    endTime = now.format('YYYY-MM-DD HH:mm:ss');
                    break;
                case 'yesterday':
                    startTime = now.subtract(1, 'day').startOf('day').format('YYYY-MM-DD HH:mm:ss');
                    endTime = now.subtract(1, 'day').endOf('day').format('YYYY-MM-DD HH:mm:ss');
                    break;
                case 'last3days':
                    startTime = now.subtract(3, 'day').startOf('day').format('YYYY-MM-DD HH:mm:ss');
                    endTime = now.format('YYYY-MM-DD HH:mm:ss');
                    break;
                case 'custom':
                    if (customTimeRange && customTimeRange[0] && customTimeRange[1]) {
                        startTime = customTimeRange[0].format('YYYY-MM-DD HH:mm:ss');
                        endTime = customTimeRange[1].format('YYYY-MM-DD HH:mm:ss');
                    } else {
                        message.warning('请选择自定义时间范围');
                        setLoading(false);
                        return;
                    }
                    break;
                default:
                    startTime = now.subtract(12, 'hour').format('YYYY-MM-DD HH:mm:ss');
                    endTime = now.format('YYYY-MM-DD HH:mm:ss');
            }

            const response = await api.getDeviceTrack(deviceId, startTime, endTime);

            if (response.data && response.data.length > 0) {
                // 将API返回的TrackPoint转换为前端格式
                const trackPoints = response.data.map((point: any) => ({
                    latitude: point.latitude,
                    longitude: point.longitude,
                    speed: point.speed || 0,
                    timestamp: new Date(point.locateTime * 1000).toISOString(),
                    temperature: point.temperature || 0,
                }));
                setTrackData(trackPoints);
                setCurrentTrackIndex(0);
                setIsPlaying(false);
                // 显示实际查询的时间范围
                const timeRangeLabel = trackTimeRange === 'today' ? '今天' :
                    trackTimeRange === 'yesterday' ? '昨天' :
                        trackTimeRange === 'last3days' ? '最近3天' : '自定义时间';
                message.success(`已加载设备 ${trackDeviceId} 的轨迹（${timeRangeLabel}，共 ${trackPoints.length} 个点）`);
            } else {
                message.warning('该设备在指定时间范围内没有轨迹数据');
                setTrackData([]);
            }
        } catch (error: any) {
            console.error('Failed to load track:', error);
            message.error(error.response?.data?.error || '加载轨迹失败');
        } finally {
            setLoading(false);
        }
    };

    const handleDeviceClick = (device: DeviceData) => {
        setFocusDevice(device);
        setViewMode('map');
    };


    const columns: ColumnsType<DeviceData> = [
        {
            title: '序号',
            key: 'index',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_: unknown, __: DeviceData, index: number) => index + 1,
        },
        {
            title: '操作',
            key: 'action',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_: unknown, record: DeviceData) => {
                const menuItems: MenuProps['items'] = [
                    {
                        key: 'view',
                        icon: <EyeOutlined />,
                        label: '设备详情',
                        onClick: () => handleDeviceClick(record),
                    },
                    {
                        key: 'track',
                        icon: <HistoryOutlined />,
                        label: '轨迹回放',
                        onClick: () => {
                            setTrackDeviceId(record.external_device_id);
                            setViewMode('track');
                        },
                    },
                    {
                        key: 'locate',
                        icon: <EnvironmentOutlined />,
                        label: '定位查看',
                        onClick: () => handleDeviceClick(record),
                    },
                    {
                        key: 'tracking',
                        icon: <GlobalOutlined />,
                        label: '货物追踪',
                        disabled: !record.bound_shipment_id,
                        onClick: () => {
                            if (record.bound_shipment_id) {
                                navigate('/business/tracking', { state: { shipmentId: record.bound_shipment_id } });
                            } else {
                                message.info('该设备未绑定运单');
                            }
                        },
                    },
                    { type: 'divider' },
                    ...(canEdit ? [
                        {
                            key: 'edit',
                            icon: <EditOutlined />,
                            label: '编辑设备',
                            onClick: () => handleEdit(record),
                        },
                        {
                            key: 'delete',
                            icon: <DeleteOutlined />,
                            label: '删除设备',
                            danger: true,
                            onClick: () => {
                                Modal.confirm({
                                    title: '确认删除',
                                    content: `确定要删除设备 ${record.name} 吗？`,
                                    okText: '删除',
                                    okType: 'danger',
                                    onOk: () => handleDelete(record.id),
                                });
                            },
                        },
                    ] : []),
                ];

                return (
                    <Dropdown
                        menu={{ items: menuItems }}
                        trigger={['hover']}
                        placement="bottomRight"
                    >
                        <Button
                            type="text"
                            icon={<SettingOutlined style={{ fontSize: 16, color: '#6b7280' }} />}
                            style={{ padding: 4 }}
                        />
                    </Dropdown>
                );
            },
        },
        {
            title: '设备号',
            dataIndex: 'external_device_id',
            key: 'external_device_id',
            width: 180,
            render: (value: string, record: DeviceData) => (
                <a onClick={() => handleDeviceClick(record)} style={{ color: '#1890ff', cursor: 'pointer', whiteSpace: 'nowrap' }}>
                    {value || '-'}
                </a>
            ),
        },
        {
            title: '使用状态',
            dataIndex: 'binding_status',
            key: 'binding_status',
            width: 100,
            render: (value: string) => (
                <Tag color={value === 'bound' ? 'blue' : 'default'}>
                    {value === 'bound' ? '已绑定' : '未绑定'}
                </Tag>
            ),
        },
        {
            title: '运输类型',
            dataIndex: 'bound_transport_type',
            key: 'bound_transport_type',
            width: 100,
            render: (value: string) => {
                if (!value) {
                    return <Tag color="default">-</Tag>;
                }
                const typeMap: Record<string, { label: string; color: string }> = {
                    sea: { label: '海运', color: 'blue' },
                    air: { label: '空运', color: 'cyan' },
                    land: { label: '陆运', color: 'green' },
                    rail: { label: '铁路', color: 'orange' },
                    multimodal: { label: '多式联运', color: 'purple' },
                };
                const config = typeMap[value] || { label: value, color: 'default' };
                return <Tag color={config.color}>{config.label}</Tag>;
            },
        },
        {
            title: '货物名称',
            dataIndex: 'bound_cargo_name',
            key: 'bound_cargo_name',
            width: 150,
            render: (value: string) => value || '-',
        },
        {
            title: '设备型号',
            dataIndex: 'type',
            key: 'type',
            width: 120,
            render: (value: string) => value || '-',
        },
        {
            title: '设备状态',
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (value: string) => (
                <Tag color={value === 'online' ? 'green' : 'default'}>
                    {value === 'online' ? '在线' : '离线'}
                </Tag>
            ),
        },
        {
            title: '定位模式',
            dataIndex: 'locate_type',
            key: 'locate_type',
            width: 100,
            render: (value: number | undefined) => {
                if (value === undefined || value === null) return '-';
                const typeMap: Record<number, string> = {
                    0: '基站定位',
                    1: 'GPS定位',
                    2: '北斗定位',
                    3: 'WIFI定位',
                };
                return typeMap[value] || `模式${value}`;
            },
        },
        {
            title: '定位类型',
            key: 'loc_type_placeholder',
            width: 100,
            render: () => '-',
        },
        {
            title: '定位周期',
            key: 'loc_period_placeholder',
            width: 100,
            render: () => '-',
        },
        {
            title: '电量',
            dataIndex: 'battery',
            key: 'battery',
            width: 80,
            render: (value: number) => {
                const battery = value || 0;
                const color = battery > 50 ? 'green' : battery > 20 ? 'orange' : 'red';
                return <Tag color={color}>{battery}%</Tag>;
            },
        },
        {
            title: '剩余工作时长',
            key: 'remaining_time_placeholder',
            width: 120,
            render: () => '-',
        },
        {
            title: '充电状态',
            key: 'charging_status_placeholder',
            width: 100,
            render: () => '-',
        },
        {
            title: '温度',
            dataIndex: 'temperature',
            key: 'temperature',
            width: 80,
            render: (value: number) => value ? `${value.toFixed(1)}°C` : '-',
        },
        {
            title: '湿度',
            dataIndex: 'humidity',
            key: 'humidity',
            width: 80,
            render: (value: number) => value ? `${value.toFixed(1)}%` : '-',
        },
        {
            title: '最后一次定位时间',
            dataIndex: 'last_update',
            key: 'last_update',
            width: 170,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
        {
            title: '当前位置',
            key: 'current_location',
            width: 180,
            render: (_: unknown, record: DeviceData) => {
                if (record.longitude && record.latitude) {
                    return `[${record.longitude.toFixed(6)}, ${record.latitude.toFixed(6)}]`;
                }
                return '-';
            },
        },
        {
            title: '供应商',
            dataIndex: 'provider',
            key: 'provider',
            width: 100,
            render: (value: string) => {
                const provider = value || 'unknown';
                const config = PROVIDERS[provider] || { label: provider, color: 'default' };
                return <Tag color={config.color}>{config.label}</Tag>;
            },
        },
        {
            title: '组织机构',
            dataIndex: 'org_name',
            key: 'org_name',
            width: 200,
            ellipsis: true,
            render: (value: string) => value || <span style={{ color: '#999' }}>-</span>,
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 170,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
        {
            title: '到期时间',
            dataIndex: 'service_end_at',
            key: 'service_end_at',
            width: 170,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
    ];

    const filteredDevices = devices.filter(d => {
        // 搜索筛选 - 支持设备名称、ID、外部设备ID（小黑卡号）
        const searchLower = search.toLowerCase();
        const matchSearch = !search ||
            (d.name || '').toLowerCase().includes(searchLower) ||
            (d.id || '').toLowerCase().includes(searchLower) ||
            (d.external_device_id || '').toLowerCase().includes(searchLower);
        // 运输类型筛选
        const matchTransportType = !filterTransportType || d.bound_transport_type === filterTransportType;
        // 使用状态筛选
        const matchBindingStatus = !filterBindingStatus || d.binding_status === filterBindingStatus;

        return matchSearch && matchTransportType && matchBindingStatus;
    });

    // Excel导出功能
    const handleExportExcel = () => {
        if (filteredDevices.length === 0) {
            message.warning('没有可导出的数据');
            return;
        }

        const statusMap: Record<string, string> = {
            online: '在线',
            offline: '离线',
        };
        const bindingStatusMap: Record<string, string> = {
            bound: '已绑定',
            unbound: '未绑定',
        };
        const transportTypeMap: Record<string, string> = {
            sea: '海运',
            air: '空运',
            land: '陆运',
            rail: '铁路',
            multimodal: '多式联运',
        };

        const excelData = filteredDevices.map((d, index) => ({
            '序号': index + 1,
            '设备号': d.external_device_id || '',
            '使用状态': bindingStatusMap[d.binding_status || ''] || '未绑定',
            '运输类型': transportTypeMap[d.bound_transport_type || ''] || '-',
            '货物名称': d.bound_cargo_name || '-',
            '设备型号': d.type || '-',
            '设备状态': statusMap[d.status] || d.status || '-',
            '定位模式': d.locate_type !== undefined ? d.locate_type : '-',
            '定位类型': '-',
            '定位周期': '-',
            '电量(%)': d.battery || 0,
            '剩余工作时长': '-',
            '充电状态': '-',
            '温度(°C)': d.temperature ? d.temperature.toFixed(1) : '-',
            '湿度(%)': d.humidity ? d.humidity.toFixed(1) : '-',
            '最后一次定位时间': d.last_update ? new Date(d.last_update).toLocaleString('zh-CN') : '-',
            '当前位置': (d.longitude && d.latitude) ? `[${d.longitude.toFixed(6)}, ${d.latitude.toFixed(6)}]` : '-',
            '供应商': PROVIDERS[d.provider]?.label || d.provider || '-',
            '组织机构': d.org_name || '-',
            '创建时间': d.created_at ? new Date(d.created_at).toLocaleString('zh-CN') : '-',
            '到期时间': d.service_end_at ? new Date(d.service_end_at).toLocaleString('zh-CN') : '-',
        }));

        const ws = XLSX.utils.json_to_sheet(excelData);
        const wb = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(wb, ws, '设备列表');

        ws['!cols'] = [
            { wch: 6 },   // 序号
            { wch: 18 },  // 设备号
            { wch: 10 },  // 使用状态
            { wch: 12 },  // 运输类型
            { wch: 18 },  // 货物名称
            { wch: 15 },  // 设备型号
            { wch: 10 },  // 设备状态
            { wch: 10 },  // 定位模式
            { wch: 10 },  // 定位类型
            { wch: 10 },  // 定位周期
            { wch: 8 },   // 电量
            { wch: 12 },  // 剩余工作时长
            { wch: 10 },  // 充电状态
            { wch: 10 },  // 温度
            { wch: 10 },  // 湿度
            { wch: 18 },  // 最后一次定位时间
            { wch: 20 },  // 当前位置
            { wch: 10 },  // 供应商
            { wch: 20 },  // 组织机构
            { wch: 18 },  // 创建时间
            { wch: 18 },  // 到期时间
        ];

        // 生成Excel文件并下载
        const fileName = `devices_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`;
        const excelBuffer = XLSX.write(wb, { bookType: 'xlsx', type: 'array' });
        downloadExcel(excelBuffer, fileName);

        message.success(`成功导出 ${filteredDevices.length} 台设备数据`);
    };

    // 列表视图
    const renderListView = () => (
        <Table
            columns={columns}
            dataSource={filteredDevices}
            rowKey="id"
            loading={loading}
            pagination={{
                defaultPageSize: 50,
                pageSizeOptions: ['10', '20', '50', '100'],
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                        <Button
                            type="default"
                            size="small"
                            icon={<DownloadOutlined />}
                            onClick={handleExportExcel}
                        >
                            导出Excel
                        </Button>
                        <span>第 {range[0]}-{range[1]} 条，共 {total} 条</span>
                    </div>
                )
            }}
            scroll={{ x: 1500, y: 'calc(100vh - 320px)' }}
            size="small"
        />
    );

    // 地图视图
    const renderMapView = () => (
        <div style={{ height: 'calc(100vh - 280px)', borderRadius: 8, overflow: 'hidden' }}>
            <Suspense fallback={<div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><Spin size="large" /></div>}>
                <DeviceMap devices={filteredDevices} providers={PROVIDERS} focusDevice={focusDevice} onClearFocus={() => setFocusDevice(null)} />
            </Suspense>
        </div>
    );

    // 格式化时间显示
    const formatTrackTime = (index: number) => {
        if (trackData.length === 0) return '00:00';
        const totalSeconds = index * 10; // 假设每个点间隔10秒
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
    };

    const getTotalTime = () => {
        if (trackData.length === 0) return '00:00';
        const totalSeconds = (trackData.length - 1) * 10;
        const minutes = Math.floor(totalSeconds / 60);
        const seconds = totalSeconds % 60;
        return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
    };

    // 轨迹回放视图
    const renderTrackView = () => (
        <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 280px)' }}>
            {/* 顶部控制栏 */}
            <div style={{ padding: 16, background: '#fafafa', borderRadius: '8px 8px 0 0' }}>
                <Space wrap>
                    <Input
                        placeholder="输入设备号"
                        value={trackDeviceId}
                        onChange={(e) => setTrackDeviceId(e.target.value)}
                        style={{ width: 200 }}
                        prefix={<SearchOutlined />}
                    />
                    {/* 时间快速选择 */}
                    <Radio.Group
                        value={trackTimeRange}
                        onChange={(e) => {
                            setTrackTimeRange(e.target.value);
                            if (e.target.value === 'custom') {
                                setCustomPickerVisible(true);
                            }
                        }}
                        optionType="button"
                        buttonStyle="solid"
                        size="middle"
                    >
                        <Radio.Button value="today">今天</Radio.Button>
                        <Radio.Button value="yesterday">昨天</Radio.Button>
                        <Radio.Button value="last3days">最近3天</Radio.Button>
                        <Radio.Button value="custom">自定义</Radio.Button>
                    </Radio.Group>
                    <Button type="primary" onClick={handleLoadTrack}>加载轨迹</Button>
                </Space>
            </div>

            {/* 地图区域 */}
            <div style={{ flex: 1, position: 'relative', minHeight: 400 }}>
                {trackData.length > 0 ? (
                    <Suspense fallback={<div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><Spin size="large" /></div>}>
                        <TrackMap trackData={trackData} currentIndex={currentTrackIndex} />
                    </Suspense>
                ) : (
                    <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
                        <Empty description="请输入设备号并加载轨迹数据" />
                    </div>
                )}
            </div>

            {/* 底部简洁播放条 */}
            {trackData.length > 0 && (
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: '12px 16px',
                    background: '#f5f5f5',
                    borderRadius: '0 0 8px 8px',
                    gap: 16,
                    borderTop: '1px solid #e8e8e8',
                }}>
                    {/* 播放按钮 */}
                    <Button
                        type="text"
                        shape="circle"
                        size="large"
                        icon={isPlaying ? <PauseOutlined style={{ fontSize: 20, color: '#1890ff' }} /> : <PlayCircleOutlined style={{ fontSize: 24, color: '#1890ff' }} />}
                        onClick={() => setIsPlaying(!isPlaying)}
                        style={{ background: 'transparent', border: 'none' }}
                    />

                    {/* 当前时间 */}
                    <span style={{ color: '#666', fontSize: 13, minWidth: 45 }}>
                        {formatTrackTime(currentTrackIndex)}
                    </span>

                    {/* 进度条 */}
                    <Slider
                        min={0}
                        max={trackData.length - 1}
                        value={currentTrackIndex}
                        onChange={(value) => setCurrentTrackIndex(value)}
                        style={{ flex: 1 }}
                        tooltip={{ formatter: (value) => formatTrackTime(value || 0) }}
                    />

                    {/* 总时间 */}
                    <span style={{ color: '#666', fontSize: 13, minWidth: 45 }}>
                        {getTotalTime()}
                    </span>

                    {/* 速度选择 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginLeft: 8 }}>
                        <span style={{ color: '#666', fontSize: 12 }}>速度:</span>
                        {[1, 2, 5].map((speed) => (
                            <Button
                                key={speed}
                                type={playSpeed === speed ? 'primary' : 'default'}
                                size="small"
                                onClick={() => setPlaySpeed(speed)}
                                style={{ minWidth: 36, fontSize: 12 }}
                            >
                                {speed}x
                            </Button>
                        ))}
                    </div>
                </div>
            )}

            {/* 自定义时间选择弹窗 */}
            <Modal
                title="选择时间范围"
                open={customPickerVisible}
                onCancel={() => setCustomPickerVisible(false)}
                onOk={() => setCustomPickerVisible(false)}
                width={400}
            >
                <DatePicker.RangePicker
                    showTime
                    value={customTimeRange}
                    onChange={(dates) => setCustomTimeRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
                    disabledDate={(current) => current && current < dayjs().subtract(6, 'month')}
                    style={{ width: '100%' }}
                    placeholder={['开始时间', '结束时间']}
                />
                <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                    * 最大查询范围：6个月内
                </div>
            </Modal>
        </div>
    );

    const tabItems = [
        { key: 'list', label: <span><UnorderedListOutlined /> 列表模式</span> },
        { key: 'map', label: <span><GlobalOutlined /> 地图模式</span> },
        { key: 'track', label: <span><PlayCircleOutlined /> 轨迹回放</span> },
    ];

    return (
        <Card
            title="设备管理"
            headStyle={{ fontSize: 16, fontWeight: 600 }}
            extra={
                <Space>
                    <Input
                        placeholder="搜索设备号"
                        prefix={<SearchOutlined />}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        style={{ width: 220 }}
                    />
                    <Select
                        placeholder="运输类型"
                        value={filterTransportType || 'all'}
                        onChange={(v) => setFilterTransportType(v === 'all' ? '' : v)}
                        style={{ width: 110 }}
                    >
                        <Select.Option value="all">全部类型</Select.Option>
                        <Select.Option value="sea">海运</Select.Option>
                        <Select.Option value="air">空运</Select.Option>
                        <Select.Option value="land">陆运</Select.Option>
                        <Select.Option value="multimodal">多式联运</Select.Option>
                    </Select>
                    <Select
                        placeholder="使用状态"
                        value={filterBindingStatus || 'all'}
                        onChange={(v) => setFilterBindingStatus(v === 'all' ? '' : v)}
                        style={{ width: 110 }}
                    >
                        <Select.Option value="all">全部状态</Select.Option>
                        <Select.Option value="bound">已绑定</Select.Option>
                        <Select.Option value="unbound">未绑定</Select.Option>
                    </Select>
                    <Select
                        value={filterOrgId || 'all'}
                        onChange={(v) => setFilterOrgId(v === 'all' ? '' : v)}
                        style={{ width: 180 }}
                        placeholder="组织机构"
                    >
                        <Select.Option value="all">全部组织</Select.Option>
                        {flatOrganizations.map(org => (
                            <Select.Option key={org.id} value={org.id}>
                                {org.level > 0 ? '└ '.repeat(org.level) : ''}{org.name}
                            </Select.Option>
                        ))}
                    </Select>
                    <Button
                        icon={<ReloadOutlined spin={syncing} />}
                        onClick={syncDevicesInBackground}
                        disabled={syncing}
                    >
                        {syncing ? '同步中' : '同步'}
                    </Button>
                    {canEdit && <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>添加设备</Button>}
                </Space>
            }
        >
            <Tabs activeKey={viewMode} onChange={(key) => setViewMode(key as ViewMode)} items={tabItems} style={{ marginTop: -8 }} />

            <div style={{ marginTop: 16 }}>
                {viewMode === 'list' && renderListView()}
                {viewMode === 'map' && renderMapView()}
                {viewMode === 'track' && renderTrackView()}
            </div>

            <Modal
                title={editingDevice ? '编辑设备' : '添加设备'}
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={() => form.submit()}
            >
                <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ provider: 'kuaihuoyun' }}>
                    <Form.Item name="provider" label="设备厂家" rules={[{ required: true, message: '请选择设备厂家' }]}>
                        <Select placeholder="请选择设备厂家">
                            <Select.Option value="kuaihuoyun">快货运</Select.Option>
                            <Select.Option value="sinoiov">中交兴路</Select.Option>
                        </Select>
                    </Form.Item>
                    <Form.Item name="external_device_id" label="设备号" rules={[{ required: true, message: '请输入设备号' }]}>
                        <Input placeholder="例如: 868120342395412" />
                    </Form.Item>
                </Form>
            </Modal>
        </Card>
    );
};

export default Devices;
