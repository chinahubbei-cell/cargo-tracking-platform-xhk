import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Tag, Button, Space, Input, Modal, Form, Select, message, Dropdown, Progress, Divider, Row, Col, DatePicker, InputNumber, Descriptions, Tabs, AutoComplete } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, SearchOutlined, DeleteOutlined, EditOutlined, EyeOutlined, SettingOutlined, EnvironmentOutlined, FileTextOutlined, SendOutlined, CheckCircleOutlined, CloseCircleOutlined, DownloadOutlined, NodeIndexOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom';
import * as XLSX from 'xlsx';
import { downloadExcel } from '../utils/downloadUtils';
import dayjs from 'dayjs';
import api from '../api/client';
import type { Shipment, Device, OrganizationTreeNode, Customer } from '../types';
import { useAuthStore } from '../store/authStore';
import { useCurrencyStore } from '../store/currencyStore';
import ShipmentLogTimeline from '../components/ShipmentLogTimeline';
import { useShipmentFieldConfig } from '../hooks/useShipmentFieldConfig';
// AddressAutocomplete 用于后续开发，暂时加下划线忽略

import AddressInput from '../components/AddressInput';
import type { AddressData } from '../components/AddressInput';
import TransportStageModal from '../components/TransportStageModal';
import styles from './Shipments.module.css';

const Shipments: React.FC = () => {
    const [searchParams] = useSearchParams();
    const [loading, setLoading] = useState(false);
    const [shipments, setShipments] = useState<Shipment[]>([]);
    const [devices, setDevices] = useState<Device[]>([]);
    const [organizations, setOrganizations] = useState<OrganizationTreeNode[]>([]);
    const [search, setSearch] = useState(searchParams.get('search') || '');
    const [filterTransportType, setFilterTransportType] = useState<string>('');
    const [filterStatus, setFilterStatus] = useState<string>('');
    const [filterOrgId, setFilterOrgId] = useState<string>('');
    const [modalVisible, setModalVisible] = useState(false);
    const [editingShipment, setEditingShipment] = useState<Shipment | null>(null);
    const [originLocation, setOriginLocation] = useState<{ lat: number; lng: number } | null>(null);
    const [destLocation, setDestLocation] = useState<{ lat: number; lng: number } | null>(null);
    const [detailModalVisible, setDetailModalVisible] = useState(false);
    const [detailActiveTab, setDetailActiveTab] = useState<string>('info');
    const [viewingShipment, setViewingShipment] = useState<Shipment | null>(null);
    const [form] = Form.useForm();
    const [stageModalShipment, setStageModalShipment] = useState<Shipment | null>(null);
    const { user } = useAuthStore();
    const { getCurrencyConfig, formatAmount } = useCurrencyStore();
    const currencyConfig = getCurrencyConfig();
    const { isFieldVisible } = useShipmentFieldConfig();
    const navigate = useNavigate();
    const location = useLocation();

    // 监听URL search参数变化
    useEffect(() => {
        const urlSearch = searchParams.get('search');
        if (urlSearch && urlSearch !== search) {
            setSearch(urlSearch);
        }
    }, [searchParams]);

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

    const isFirstRender = React.useRef(true);
    // 客户自动补全状态
    const [senderOptions, setSenderOptions] = useState<{ value: string; label: string; customer: Customer }[]>([]);
    const [receiverOptions, setReceiverOptions] = useState<{ value: string; label: string; customer: Customer }[]>([]);

    const handleCustomerSearch = async (value: string, type: 'sender' | 'receiver') => {
        console.log('Search triggered:', value, type);
        if (!value || value.length < 3) {
            console.log('Value too short, clearing options');
            if (type === 'sender') setSenderOptions([]);
            else setReceiverOptions([]);
            return;
        }
        try {
            // 后端现在返回模糊搜索结果数组
            const res = await api.searchCustomers({ phone: value, type });
            console.log('Search API response:', res);

            if (res.data && Array.isArray(res.data)) {
                const customers = res.data as Customer[];
                console.log('Customers found:', customers);
                const options = customers.map(customer => ({
                    value: customer.phone,
                    label: `${customer.name} - ${customer.phone}${customer.company ? ' (' + customer.company + ')' : ''}`,
                    customer
                }));
                console.log('Generated options:', options);
                if (type === 'sender') setSenderOptions(options);
                else setReceiverOptions(options);
            } else {
                console.log('No data or invalid format:', res);
                if (type === 'sender') setSenderOptions([]);
                else setReceiverOptions([]);
            }
        } catch (error) {
            console.error('Search error:', error);
            // 忽略错误
            if (type === 'sender') setSenderOptions([]);
            else setReceiverOptions([]);
        }
    };

    const onCustomerSelect = async (_: string, option: any, type: 'sender' | 'receiver') => {
        const customer = option.customer as Customer;
        if (type === 'sender') {
            form.setFieldsValue({
                sender_name: customer.name,
                origin_address: customer.address || form.getFieldValue('origin_address')
            });
            // 自动解析发货地址坐标
            if (customer.address) {
                try {
                    const res = await api.geocode(customer.address);
                    if (res.success && res.data) {
                        setOriginLocation({ lat: res.data.lat, lng: res.data.lng });
                        // 同时也填充发货城市/地点名
                        if (!form.getFieldValue('origin')) {
                            form.setFieldsValue({ origin: res.data.short_name || res.data.city || customer.city });
                        }
                        message.success('已自动获取发货地址坐标');
                    } else {
                        message.warning('无法解析发货地址坐标，请手动选择地址');
                    }
                } catch (e: any) {
                    console.error('Geocode failed:', e);
                    message.warning('地址解析失败: ' + (e.response?.data?.error || e.message || '未知错误'));
                }
            } else {
                message.warning('该客户未设置地址，请手动输入地址');
            }
        } else {
            form.setFieldsValue({
                receiver_name: customer.name,
                dest_address: customer.address || form.getFieldValue('dest_address')
            });
            // 自动解析收货地址坐标
            if (customer.address) {
                try {
                    const res = await api.geocode(customer.address);
                    if (res.success && res.data) {
                        setDestLocation({ lat: res.data.lat, lng: res.data.lng });
                        // 同时也填充收货城市/地点名
                        if (!form.getFieldValue('destination')) {
                            form.setFieldsValue({ destination: res.data.short_name || res.data.city || customer.city });
                        }
                        message.success('已自动获取收货地址坐标');
                    } else {
                        message.warning('无法解析收货地址坐标，请手动选择地址');
                    }
                } catch (e: any) {
                    console.error('Geocode failed:', e);
                    message.warning('地址解析失败: ' + (e.response?.data?.error || e.message || '未知错误'));
                }
            } else {
                message.warning('该客户未设置地址，请手动输入地址');
            }
        }
    };

    // 模拟运单数据（使用新运单号格式）
    const mockShipments: Shipment[] = [
        { id: '260116000001', origin: '上海', destination: '北京', status: 'in_transit', progress: 65, eta: new Date(Date.now() + 86400000 * 2).toISOString(), created_at: new Date().toISOString() },
        { id: '260116000002', origin: '深圳', destination: '广州', status: 'delivered', progress: 100, eta: new Date().toISOString(), created_at: new Date(Date.now() - 86400000).toISOString() },
        { id: '260116000003', origin: '杭州', destination: '武汉', status: 'pending', progress: 0, eta: new Date(Date.now() + 86400000 * 5).toISOString(), created_at: new Date().toISOString() },
        { id: '260116000004', origin: '成都', destination: '重庆', status: 'in_transit', progress: 35, eta: new Date(Date.now() + 86400000 * 3).toISOString(), created_at: new Date().toISOString() },
    ];

    const loadData = useCallback(async () => {
        setLoading(true);
        const isDevelopment = import.meta.env.DEV;
        try {
            const [shipmentsRes, devicesRes, orgsRes] = await Promise.all([
                api.getShipments({ search, org_id: filterOrgId }),
                api.getDevices({}),
                api.getOrganizations({ tree: true }),
            ]);
            // 生产环境不使用模拟数据，开发环境可使用模拟数据
            if (shipmentsRes.data && shipmentsRes.data.length > 0) {
                console.log('LoadData Success:', shipmentsRes.data.length, 'shipments');
                setShipments(shipmentsRes.data);
            } else if (isDevelopment) {
                console.log('LoadData Empty/Failed, using Mock data');
                setShipments(mockShipments);
            } else {
                console.log('LoadData Empty, setting empty list');
                setShipments([]);
            }
            if (devicesRes.data) setDevices(devicesRes.data);
            if (orgsRes) setOrganizations(orgsRes);
        } catch (error) {
            console.error('API请求失败', error);
            if (isDevelopment) {
                setShipments(mockShipments);
            } else {
                setShipments([]);
            }
        } finally {
            setLoading(false);
        }
    }, [search, filterOrgId]);

    const handleCreate = useCallback(() => {
        setEditingShipment(null);
        form.resetFields();
        setOriginLocation(null);
        setDestLocation(null);
        setModalVisible(true);
    }, [form]);

    const handleEdit = useCallback((shipment: Shipment) => {
        setEditingShipment(shipment);

        // 转换日期字段为dayjs对象
        const formValues = {
            ...shipment,
            etd: shipment.etd ? dayjs(shipment.etd) : undefined,
            eta: shipment.eta ? dayjs(shipment.eta) : undefined,
            atd: shipment.atd ? dayjs(shipment.atd) : undefined,
        };

        form.setFieldsValue(formValues);
        // Restore location states from shipment data
        if (shipment.origin_lat && shipment.origin_lng) {
            setOriginLocation({ lat: shipment.origin_lat, lng: shipment.origin_lng });
        } else {
            setOriginLocation(null);
        }
        if (shipment.dest_lat && shipment.dest_lng) {
            setDestLocation({ lat: shipment.dest_lat, lng: shipment.dest_lng });
        } else {
            setDestLocation(null);
        }
        setModalVisible(true);
    }, [form]);

    const handleDelete = useCallback(async (id: string) => {
        try {
            await api.deleteShipment(id);
            message.success('删除成功');
            loadData();
        } catch (error) {
            message.error('删除失败');
        }
    }, [loadData]);

    const handleViewDetail = useCallback((shipment: Shipment, activeTab: string = 'info') => {
        setViewingShipment(shipment);
        setDetailActiveTab(activeTab);
        setDetailModalVisible(true);
    }, []);

    useEffect(() => {
        loadData();
    }, []);

    // 组织筛选变化时重新加载数据
    useEffect(() => {
        // 跳过首次渲染
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        loadData();
    }, [filterOrgId]);

    const [originPortCode, setOriginPortCode] = useState<string>('');
    const [destPortCode, setDestPortCode] = useState<string>('');

    // ... (keep existing codes)

    // 从线路规划页面跳转过来时，自动打开创建对话框
    useEffect(() => {
        const state = location.state as {
            fromRoutePlanning?: boolean;
            origin?: string;
            destination?: string;
            origin_address?: string;
            dest_address?: string;
            origin_lat?: number;
            origin_lng?: number;
            dest_lat?: number;
            dest_lng?: number;
            routePlan?: {
                type: string;
                label: string;
                total_days: number;
                total_cost: number;
                segments?: {
                    type: string;
                    from: string;
                    to: string;
                    mode: string;
                    carrier?: string;
                }[];
            };
            transport_type?: string;
            transport_mode?: string;
            container_type?: string;
            weight_kg?: number;
            volume_cbm?: number;
            quantity?: number;
            cargo_type?: string;
            freight_cost?: number;
            total_days?: number;
            etd?: string;
            eta?: string;
            carrier?: string;
            currency?: string;
        } | null;
        if (state?.fromRoutePlanning) {
            setEditingShipment(null);
            form.resetFields();
            setOriginPortCode('');
            setDestPortCode('');

            // 使用直接传递的transport_type，或从segments推断
            let transportType = state.transport_type || 'multimodal';
            if (!state.transport_type && state.routePlan?.segments && state.routePlan.segments.length > 0) {
                const modes = state.routePlan.segments.map(s => s.mode);
                if (modes.includes('ocean')) transportType = 'sea';
                else if (modes.includes('air')) transportType = 'air';
                else if (modes.every(m => m === 'truck' || m === 'rail')) transportType = 'land';
            }

            // 提取港口代码 (从 line_haul 环节提取)
            if (state.routePlan?.segments) {
                const lineHaul = state.routePlan.segments.find(s => s.type === 'line_haul');
                if (lineHaul) {
                    // 格式: CODE (NAME)
                    const extractCode = (str: string) => str.split(' ')[0];
                    const opCode = extractCode(lineHaul.from);
                    const dpCode = extractCode(lineHaul.to);
                    setOriginPortCode(opCode);
                    setDestPortCode(dpCode);
                    console.log('Detected Ports from Route Plan:', opCode, dpCode);
                }
            }

            // 设置坐标 - 从线路规划传递过来
            if (state.origin_lat && state.origin_lng) {
                setOriginLocation({ lat: state.origin_lat, lng: state.origin_lng });
                console.log('Set origin coordinates from route planning:', state.origin_lat, state.origin_lng);
            } else {
                setOriginLocation(null);
            }
            if (state.dest_lat && state.dest_lng) {
                setDestLocation({ lat: state.dest_lat, lng: state.dest_lng });
                console.log('Set destination coordinates from route planning:', state.dest_lat, state.dest_lng);
            } else {
                setDestLocation(null);
            }

            // 预填发货地、目的地和线路规划数据 - 全部字段
            form.setFieldsValue({
                // 地址
                origin: state.origin || '',
                destination: state.destination || '',
                origin_address: state.origin_address || state.origin || '',
                dest_address: state.dest_address || state.destination || '',
                // 运输参数
                transport_type: transportType,
                transport_mode: state.transport_mode || 'lcl',
                container_type: state.container_type || '20GP',
                // 货物参数
                weight: state.weight_kg || undefined,
                volume: state.volume_cbm || undefined,
                pieces: state.quantity || undefined,
                cargo_type: state.cargo_type || 'general',
                // 时间 - 转换为 dayjs 对象供 DatePicker 使用
                etd: state.etd ? dayjs(state.etd) : undefined,
                eta: state.eta ? dayjs(state.eta) : undefined,
                // 费用
                freight_cost: state.freight_cost || undefined,
                // 承运商
                carrier: state.carrier || undefined,
            });
            setModalVisible(true);
            // 清除state防止刷新后重复打开
            navigate(location.pathname, { replace: true, state: null });
        }

        // 从预警中心或其他页面跳转过来时，自动搜索运单号
        if ((state as any)?.shipmentId) {
            const shipmentId = (state as any).shipmentId;
            setSearch(shipmentId);
            // 清除state防止刷新后重复搜索
            navigate(location.pathname, { replace: true, state: null });
        }
    }, [location.state]);

    // ... (keep mockShipments and loadData)

    // ...

    const handleSubmit = useCallback(async (values: any) => {
        // Merge location data from pickers, fallback to form values if picker not used
        const payload = {
            ...values,
            // 确保清除设备时发送空字符串而非undefined
            device_id: values.device_id ?? '',
            origin_lat: originLocation?.lat ?? values.origin_lat,
            origin_lng: originLocation?.lng ?? values.origin_lng,
            dest_lat: destLocation?.lat ?? values.dest_lat,
            dest_lng: destLocation?.lng ?? values.dest_lng,
            // Global default radius (1000m) as per new requirement
            origin_radius: 1000,
            dest_radius: 1000,
            // 传递港口代码 (用于自动生成Stages)
            origin_port_code: originPortCode,
            dest_port_code: destPortCode,
            // 新建时使用当前筛选的组织（如果有）
            ...(filterOrgId ? { org_id: filterOrgId } : {}),
        };
        try {
            if (editingShipment) {
                await api.updateShipment(editingShipment.id, payload);
                message.success('更新成功');
            } else {
                await api.createShipment(payload);
                message.success('创建成功');
            }
            setModalVisible(false);
            loadData();
        } catch (error: any) {
            message.error(error.response?.data?.error || '操作失败');
        }
    }, [editingShipment, originLocation, destLocation, filterOrgId, loadData, originPortCode, destPortCode]);

    const handleViewTracking = useCallback((shipment: Shipment) => {
        navigate('/business/tracking', { state: { shipmentId: shipment.id } });
    }, [navigate]);

    // 快捷状态切换
    const handleStatusTransition = useCallback(async (shipmentId: string, action: 'depart' | 'deliver' | 'cancel', actionLabel: string) => {
        // 签收操作使用带输入框的弹窗
        if (action === 'deliver') {
            let receiverName = '';
            Modal.confirm({
                title: '确认签收',
                icon: null,
                content: (
                    <div className={styles.deliverModalContent}>
                        <p>确定要将运单 {shipmentId} 标记为签收吗？</p>
                        <div style={{ marginTop: 12 }}>
                            <label className={styles.deliverModalLabel}>签收人</label>
                            <Input
                                placeholder="请输入签收人姓名"
                                onChange={(e) => { receiverName = e.target.value; }}
                            />
                        </div>
                    </div>
                ),
                okText: '确认签收',
                cancelText: '取消',
                onOk: async () => {
                    try {
                        const res = await api.transitionShipmentStatus(shipmentId, action, receiverName);
                        message.success(res.data?.message || '签收成功');
                        loadData();
                    } catch (error: any) {
                        message.error(error.response?.data?.error || '签收失败');
                    }
                },
            });
            return;
        }

        // 其他操作使用普通确认弹窗
        Modal.confirm({
            title: `确认${actionLabel}`,
            content: `确定要将运单 ${shipmentId} 标记为${actionLabel}吗？`,
            okText: '确认',
            cancelText: '取消',
            onOk: async () => {
                try {
                    const res = await api.transitionShipmentStatus(shipmentId, action);
                    message.success(res.data?.message || `${actionLabel}成功`);
                    loadData();
                } catch (error: any) {
                    message.error(error.response?.data?.error || `${actionLabel}失败`);
                }
            },
        });
    }, [loadData]);

    // Excel导出功能
    const handleExportExcel = useCallback(() => {
        // 获取当前筛选后的数据
        const filteredData = shipments.filter(s => {
            const searchLower = search.toLowerCase();
            const matchSearch = !search ||
                (s.id || '').toLowerCase().includes(searchLower) ||
                (s.bill_of_lading || '').toLowerCase().includes(searchLower) ||
                (s.cargo_name || '').toLowerCase().includes(searchLower) ||
                (s.container_no || '').toLowerCase().includes(searchLower) ||
                (s.origin || '').toLowerCase().includes(searchLower) ||
                (s.destination || '').toLowerCase().includes(searchLower);
            const matchTransportType = !filterTransportType || s.transport_type === filterTransportType;
            const matchStatus = !filterStatus || s.status === filterStatus;
            return matchSearch && matchTransportType && matchStatus;
        });

        if (filteredData.length === 0) {
            message.warning('没有可导出的数据');
            return;
        }

        // 定义Excel列映射
        const transportTypeMap: Record<string, string> = {
            sea: '海运', air: '空运', land: '陆运', multimodal: '多式联运'
        };
        const statusMap: Record<string, string> = {
            pending: '待发货', in_transit: '运输中', delivered: '已到达', cancelled: '已取消'
        };
        const cargoTypeMap: Record<string, string> = {
            general: '普货', dangerous: '危险品', cold_chain: '冷链'
        };
        const transportModeMap: Record<string, string> = {
            lcl: '零担', fcl: '整柜'
        };

        // 转换数据为Excel格式
        const excelData = filteredData.map((s, index) => ({
            '序号': index + 1,
            '运单号': s.id || '',
            '状态': statusMap[s.status] || s.status || '',
            '进度(%)': s.progress || 0,
            '发货地': s.origin || '',
            '目的地': s.destination || '',
            '运输类型': s.transport_type ? (transportTypeMap[s.transport_type as keyof typeof transportTypeMap] || s.transport_type) : '',
            '货物名称': s.cargo_name || '',
            '货物类型': cargoTypeMap[s.cargo_type || ''] || s.cargo_type || '',
            '运输模式': transportModeMap[s.transport_mode || ''] || s.transport_mode || '',
            '柜型': s.container_type || '',
            '组织机构': s.org_name || '',
            '提单号': s.bill_of_lading || '',
            '箱号/车牌': s.container_no || '',
            '船名': s.vessel_name || '',
            '航次': s.voyage_no || '',
            '船司/航司': s.carrier || '',
            '封条号': s.seal_no || '',
            '件数': s.pieces ?? '',
            '重量(kg)': s.weight ?? '',
            '体积(m³)': s.volume ?? '',
            'PO单号': s.po_numbers || '',
            'SKU ID': s.sku_ids || '',
            'FBA编号': s.fba_shipment_id || '',
            '发货人': s.sender_name || '',
            '发货电话': s.sender_phone || '',
            '发货地址': s.origin_address || '',
            '收货人': s.receiver_name || '',
            '收货电话': s.receiver_phone || '',
            '收货地址': s.dest_address || '',
            '运费': s.freight_cost ?? '',
            '附加费': s.surcharges ?? '',
            '关税': s.customs_fee ?? '',
            '其他费用': s.other_cost ?? '',
            '总费用': s.total_cost ?? '',
            '设备ID': (() => {
                if (s.device?.external_device_id) return s.device.external_device_id;
                if (s.device_id) return s.device_id;
                if (s.unbound_device_id) {
                    const found = devices.find(d => d.id === s.unbound_device_id);
                    return found?.external_device_id || s.unbound_device_id;
                }
                return '';
            })(),
            '预计出发(ETD)': s.etd ? new Date(s.etd).toLocaleString('zh-CN') : '',
            '实际出发(ATD)': s.atd ? new Date(s.atd).toLocaleString('zh-CN') : '',
            '预计到达(ETA)': s.eta ? new Date(s.eta).toLocaleString('zh-CN') : '',
            '实际到达(ATA)': s.ata ? new Date(s.ata).toLocaleString('zh-CN') : '',
            '创建时间': s.created_at ? new Date(s.created_at).toLocaleString('zh-CN') : '',
        }));

        // 创建工作簿
        const ws = XLSX.utils.json_to_sheet(excelData);
        const wb = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(wb, ws, '运单列表');

        // 设置列宽
        ws['!cols'] = [
            { wch: 6 },   // 序号
            { wch: 14 },  // 运单号
            { wch: 10 },  // 状态
            { wch: 8 },   // 进度
            { wch: 20 },  // 发货地
            { wch: 20 },  // 目的地
            { wch: 12 },  // 运输类型
            { wch: 18 },  // 货物名称
            { wch: 10 },  // 货物类型
            { wch: 10 },  // 运输模式
            { wch: 10 },  // 柜型
            { wch: 20 },  // 组织机构
            { wch: 18 },  // 提单号
            { wch: 15 },  // 箱号
            { wch: 15 },  // 船名
            { wch: 12 },  // 航次
            { wch: 12 },  // 船司
            { wch: 12 },  // 封条号
            { wch: 8 },   // 件数
            { wch: 10 },  // 重量
            { wch: 10 },  // 体积
            { wch: 15 },  // PO单号
            { wch: 15 },  // SKU ID
            { wch: 15 },  // FBA编号
            { wch: 15 },  // 发货人
            { wch: 15 },  // 发货电话
            { wch: 30 },  // 发货地址
            { wch: 15 },  // 收货人
            { wch: 15 },  // 收货电话
            { wch: 30 },  // 收货地址
            { wch: 12 },  // 运费
            { wch: 12 },  // 附加费
            { wch: 12 },  // 关税
            { wch: 12 },  // 其他费用
            { wch: 12 },  // 总费用
            { wch: 18 },  // 设备ID
            { wch: 18 },  // ETD
            { wch: 18 },  // ATD
            { wch: 18 },  // ETA
            { wch: 18 },  // ATA
            { wch: 18 },  // 创建时间
        ];

        // 生成Excel文件并下载
        const fileName = `shipments_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`;
        const excelBuffer = XLSX.write(wb, { bookType: 'xlsx', type: 'array' });
        downloadExcel(excelBuffer, fileName);

        message.success(`成功导出 ${filteredData.length} 条运单数据`);
    }, [shipments, search, filterTransportType, filterStatus, devices]);

    const statusColors: Record<string, string> = {
        pending: '#f59e0b',
        in_transit: '#2563eb',
        delivered: '#10b981',
        cancelled: '#ef4444',
    };

    const statusLabels: Record<string, string> = {
        pending: '待发货',
        in_transit: '运输中',
        delivered: '已到达',
        cancelled: '已取消',
    };



    const columns: ColumnsType<Shipment> = React.useMemo(() => [
        {
            title: '序号',
            key: 'index',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_, __, index) => index + 1,
        },
        {
            title: '操作',
            key: 'action',
            width: 60,
            fixed: 'left',
            align: 'center' as const,
            render: (_, record) => {
                // 状态快捷操作按钮
                const statusActions: MenuProps['items'] = [];
                if (canEdit) {
                    if (record.status === 'pending') {
                        statusActions.push({
                            key: 'depart',
                            icon: <SendOutlined />,
                            label: '🚀 发车',
                            onClick: () => handleStatusTransition(record.id, 'depart', '发车'),
                        });
                    }
                    if (record.status === 'in_transit') {
                        statusActions.push({
                            key: 'deliver',
                            icon: <CheckCircleOutlined />,
                            label: '✅ 签收',
                            onClick: () => handleStatusTransition(record.id, 'deliver', '签收'),
                        });
                    }
                    if (record.status === 'pending' || record.status === 'in_transit') {
                        statusActions.push({
                            key: 'cancel',
                            icon: <CloseCircleOutlined />,
                            label: '取消运单',
                            danger: true,
                            onClick: () => handleStatusTransition(record.id, 'cancel', '取消'),
                        });
                    }
                }

                const menuItems: MenuProps['items'] = [
                    // 运输环节作为主入口
                    {
                        key: 'stage',
                        icon: <NodeIndexOutlined />,
                        label: '运输环节',
                        onClick: () => setStageModalShipment(record),
                    },
                    { type: 'divider' },
                    {
                        key: 'track',
                        icon: <EnvironmentOutlined />,
                        label: '货物追踪',
                        onClick: () => handleViewTracking(record),
                    },
                    {
                        key: 'detail',
                        icon: <EyeOutlined />,
                        label: '运单详情',
                        onClick: () => handleViewDetail(record, 'info'),
                    },
                    {
                        key: 'log',
                        icon: <FileTextOutlined />,
                        label: '查看日志',
                        onClick: () => handleViewDetail(record, 'logs'),
                    },
                    { type: 'divider' },
                    ...(canEdit ? [
                        {
                            key: 'edit',
                            icon: <EditOutlined />,
                            label: '编辑运单',
                            onClick: () => handleEdit(record),
                        },
                        // 取消运单保留在菜单中
                        ...((record.status === 'pending' || record.status === 'in_transit') ? [{
                            key: 'cancel',
                            icon: <CloseCircleOutlined />,
                            label: '取消运单',
                            danger: true,
                            onClick: () => handleStatusTransition(record.id, 'cancel', '取消'),
                        }] : []),
                        {
                            key: 'delete',
                            icon: <DeleteOutlined />,
                            label: '删除运单',
                            danger: true,
                            onClick: () => {
                                Modal.confirm({
                                    title: '确认删除',
                                    content: `确定要删除运单 ${record.id} 吗？`,
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
                            icon={<SettingOutlined className={styles.settingIcon} />}
                            className={styles.actionButton}
                        />
                    </Dropdown>
                );
            },
        },
        {
            title: '运单号',
            dataIndex: 'id',
            key: 'id',
            width: 140,
            render: (id: string, record: Shipment) => (
                <span
                    className={styles.shipmentId}
                    onClick={() => handleViewDetail(record, 'info')}
                    title="点击查看详情"
                >
                    {id}
                </span>
            ),
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (status: string) => (
                <Tag color={statusColors[status]}>{statusLabels[status] || status}</Tag>
            ),
        },
        {
            title: '进度',
            dataIndex: 'progress',
            key: 'progress',
            width: 120,
            render: (progress: number) => (
                <Progress
                    percent={progress || 0}
                    size="small"
                    status={progress === 100 ? 'success' : 'active'}
                />
            ),
        },
        {
            title: '发货地',
            dataIndex: 'origin',
            key: 'origin',
            width: 150,
            ellipsis: true,
        },
        {
            title: '目的地',
            dataIndex: 'destination',
            key: 'destination',
            width: 150,
            ellipsis: true,
        },
        {
            title: '运输类型',
            dataIndex: 'transport_type',
            key: 'transport_type',
            width: 100,
            render: (value: string) => {
                const typeMap: Record<string, { label: string; color: string }> = {
                    sea: { label: '海运', color: 'blue' },
                    air: { label: '空运', color: 'cyan' },
                    land: { label: '陆运', color: 'green' },
                    multimodal: { label: '多式联运', color: 'purple' },
                };
                const config = typeMap[value] || { label: value || '-', color: 'default' };
                return <Tag color={config.color}>{config.label}</Tag>;
            },
        },
        {
            title: '货物名称',
            dataIndex: 'cargo_name',
            key: 'cargo_name',
            width: 150,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '货物类型',
            dataIndex: 'cargo_type',
            key: 'cargo_type',
            width: 100,
            render: (value: string) => {
                const map: Record<string, string> = { general: '普货', dangerous: '危险品', cold_chain: '冷链' };
                return map[value] || value || '-';
            }
        },
        {
            title: '运输模式',
            dataIndex: 'transport_mode',
            key: 'transport_mode',
            width: 100,
            render: (value: string) => {
                const map: Record<string, string> = { lcl: '零担', fcl: '整柜' };
                return map[value] || value || '-';
            }
        },
        {
            title: '柜型',
            dataIndex: 'container_type',
            key: 'container_type',
            width: 100,
            render: (value: string) => value || '-',
        },
        {
            title: '组织机构',
            dataIndex: 'org_name',
            key: 'org_name',
            width: 200,
            ellipsis: true,
            render: (value: string) => value || <span style={{ color: '#999' }}>-</span>,
        },
        ...(isFieldVisible('bill_of_lading') ? [{
            title: '提单号',
            dataIndex: 'bill_of_lading',
            key: 'bill_of_lading',
            width: 140,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('container_no') ? [{
            title: '箱号/车牌',
            dataIndex: 'container_no',
            key: 'container_no',
            width: 130,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        // ===== 船务信息 =====
        ...(isFieldVisible('vessel_name') ? [{
            title: '船名',
            dataIndex: 'vessel_name',
            key: 'vessel_name',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('voyage_no') ? [{
            title: '航次',
            dataIndex: 'voyage_no',
            key: 'voyage_no',
            width: 100,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('carrier') ? [{
            title: '船司/航司',
            dataIndex: 'carrier',
            key: 'carrier',
            width: 100,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('seal_no') ? [{
            title: '封条号',
            dataIndex: 'seal_no',
            key: 'seal_no',
            width: 100,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        // ===== 货物量纲 =====
        {
            title: '件数',
            dataIndex: 'pieces',
            key: 'pieces',
            width: 80,
            align: 'right' as const,
            render: (value: number) => value != null ? value.toLocaleString() : '-',
        },
        {
            title: '重量(kg)',
            dataIndex: 'weight',
            key: 'weight',
            width: 100,
            align: 'right' as const,
            render: (value: number) => value != null ? value.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) : '-',
        },
        {
            title: '体积(m³)',
            dataIndex: 'volume',
            key: 'volume',
            width: 100,
            align: 'right' as const,
            render: (value: number) => value != null ? value.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) : '-',
        },
        // ===== 订单关联 =====
        ...(isFieldVisible('po_numbers') ? [{
            title: 'PO单号',
            dataIndex: 'po_numbers',
            key: 'po_numbers',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('sku_ids') ? [{
            title: 'SKU ID',
            dataIndex: 'sku_ids',
            key: 'sku_ids',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        ...(isFieldVisible('fba_shipment_id') ? [{
            title: 'FBA编号',
            dataIndex: 'fba_shipment_id',
            key: 'fba_shipment_id',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        }] : []),
        // ===== 路由信息 =====
        {
            title: '发货人',
            dataIndex: 'sender_name',
            key: 'sender_name',
            width: 100,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '发货电话',
            dataIndex: 'sender_phone',
            key: 'sender_phone',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '收货人',
            dataIndex: 'receiver_name',
            key: 'receiver_name',
            width: 100,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '收货电话',
            dataIndex: 'receiver_phone',
            key: 'receiver_phone',
            width: 120,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '发货地址',
            dataIndex: 'origin_address',
            key: 'origin_address',
            width: 200,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        {
            title: '收货地址',
            dataIndex: 'dest_address',
            key: 'dest_address',
            width: 200,
            ellipsis: true,
            render: (value: string) => value || '-',
        },
        // ===== 費用信息 =====
        {
            title: `运费(${currencyConfig.code})`,
            dataIndex: 'freight_cost',
            key: 'freight_cost',
            width: 100,
            align: 'right' as const,
            render: (value: number) => formatAmount(value),
        },
        ...(isFieldVisible('surcharges') ? [{
            title: `附加费(${currencyConfig.code})`,
            dataIndex: 'surcharges',
            key: 'surcharges',
            width: 100,
            align: 'right' as const,
            render: (value: number) => formatAmount(value),
        }] : []),
        ...(isFieldVisible('customs_fee') ? [{
            title: `关税(${currencyConfig.code})`,
            dataIndex: 'customs_fee',
            key: 'customs_fee',
            width: 100,
            align: 'right' as const,
            render: (value: number) => formatAmount(value),
        }] : []),
        ...(isFieldVisible('other_cost') ? [{
            title: `其他费用(${currencyConfig.code})`,
            dataIndex: 'other_cost',
            key: 'other_cost',
            width: 100,
            align: 'right' as const,
            render: (value: number) => formatAmount(value),
        }] : []),
        {
            title: `总费用(${currencyConfig.code})`,
            dataIndex: 'total_cost',
            key: 'total_cost',
            width: 120,
            align: 'right' as const,
            render: (value: number) => formatAmount(value),
        },
        {
            title: '定位设备ID号',
            dataIndex: 'device',
            key: 'device_id',
            width: 160,
            render: (_: unknown, record: Shipment) => {
                // 优先显示关联设备的external_device_id
                const device = record.device;
                if (device?.external_device_id) {
                    return <span className={styles.deviceId}>{device.external_device_id}</span>;
                }
                // 如果没有关联device对象，尝试从devices列表中查找
                if (record.device_id) {
                    const foundDevice = devices.find(d => d.id === record.device_id);
                    if (foundDevice?.external_device_id) {
                        return <span className={styles.deviceId}>{foundDevice.external_device_id}</span>;
                    }
                    return <span className={styles.deviceId}>{record.device_id}</span>;
                }
                // 如果已解绑，显示解绑前的设备ID
                if (record.unbound_device_id) {
                    // 尝试在devices列表中查找以显示external_device_id
                    const foundDevice = devices.find(d => d.id === record.unbound_device_id);
                    if (foundDevice?.external_device_id) {
                        return <span className={styles.deviceId}>{foundDevice.external_device_id}</span>;
                    }
                    return <span className={styles.deviceId}>{record.unbound_device_id}</span>;
                }
                return '-';
            },
        },
        // ===== 时间字段 =====
        {
            title: '预计出发(ETD)',
            dataIndex: 'etd',
            key: 'etd',
            width: 150,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
        {
            title: '实际出发(ATD)',
            dataIndex: 'atd',
            key: 'atd',
            width: 150,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
        {
            title: '预计到达(ETA)',
            dataIndex: 'eta',
            key: 'eta',
            width: 150,
            render: (eta: string) => eta ? new Date(eta).toLocaleString('zh-CN') : '-',
        },
        {
            title: '实际到达(ATA)',
            dataIndex: 'ata',
            key: 'ata_final',
            width: 150,
            render: (value: string) => value ? new Date(value).toLocaleString('zh-CN') : '-',
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (time: string) => new Date(time).toLocaleString('zh-CN'),
        },
    ], [canEdit, devices, handleStatusTransition, handleViewDetail, handleViewTracking, handleEdit, handleDelete, formatAmount, currencyConfig, isFieldVisible]);

    return (
        <Card
            title="运单管理"
            headStyle={{ fontSize: 16, fontWeight: 600 }} // Keep headStyle as prop mostly expecting object, or use className if supported
            className={styles.cardHead} // Attempt usage if Card supports or wrap content

            extra={
                <Space>
                    <Input

                        placeholder="搜索运单"
                        prefix={<SearchOutlined />}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onPressEnter={() => loadData()}
                        className={styles.searchConfig}
                    />
                    <Select
                        value={filterStatus || 'all'}
                        onChange={(v) => setFilterStatus(v === 'all' ? '' : v)}
                        className={styles.filterSelect}
                    >
                        <Select.Option value="all">全部状态</Select.Option>
                        <Select.Option value="pending">待发货</Select.Option>
                        <Select.Option value="in_transit">运输中</Select.Option>
                        <Select.Option value="delivered">已到达</Select.Option>
                        <Select.Option value="cancelled">已取消</Select.Option>
                    </Select>
                    <Select
                        value={filterTransportType || 'all'}
                        onChange={(v) => setFilterTransportType(v === 'all' ? '' : v)}
                        className={styles.filterSelect}
                    >
                        <Select.Option value="all">全部类型</Select.Option>
                        <Select.Option value="sea">海运</Select.Option>
                        <Select.Option value="air">空运</Select.Option>
                        <Select.Option value="land">陆运</Select.Option>
                        <Select.Option value="multimodal">多式联运</Select.Option>
                    </Select>
                    <Select
                        value={filterOrgId || 'all'}
                        onChange={(v) => setFilterOrgId(v === 'all' ? '' : v)}
                        className={styles.orgSelect}
                        placeholder="组织机构"
                    >
                        <Select.Option value="all">全部组织</Select.Option>
                        {flatOrganizations.map(org => (
                            <Select.Option key={org.id} value={org.id}>
                                {org.level > 0 ? '└ '.repeat(org.level) : ''}{org.name}
                            </Select.Option>
                        ))}
                    </Select>
                    {canEdit && (
                        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
                            创建运单
                        </Button>
                    )}
                </Space>
            }
        >
            <Table
                columns={columns}
                dataSource={shipments.filter(s => {
                    const searchLower = search.toLowerCase();
                    const matchSearch = !search ||
                        (s.id || '').toLowerCase().includes(searchLower) ||
                        (s.bill_of_lading || '').toLowerCase().includes(searchLower) ||
                        (s.cargo_name || '').toLowerCase().includes(searchLower) ||
                        (s.container_no || '').toLowerCase().includes(searchLower) ||
                        (s.origin || '').toLowerCase().includes(searchLower) ||
                        (s.destination || '').toLowerCase().includes(searchLower);
                    const matchTransportType = !filterTransportType || s.transport_type === filterTransportType;
                    const matchStatus = !filterStatus || s.status === filterStatus;
                    return matchSearch && matchTransportType && matchStatus;
                })}
                rowKey="id"
                loading={loading}
                pagination={{
                    defaultPageSize: 50,
                    pageSizeOptions: ['10', '20', '50', '100'],
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total, range) => (
                        <div className={styles.paginationContainer}>
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
                scroll={{ x: 3500, y: 'calc(100vh - 280px)' }}
                size="small"
                locale={{
                    emptyText: filterOrgId && search ? (
                        <div className={styles.emptyContainer}>
                            <p className={styles.emptyTitle}>在当前组织下未找到该运单</p>
                            <p className={styles.emptyDesc}>尝试选择"全部组织"或切换到运单所属的组织进行搜索</p>
                        </div>
                    ) : filterOrgId ? (
                        <div style={{ padding: 40, textAlign: 'center', color: '#999' }}>
                            <p>当前组织下暂无运单</p>
                        </div>
                    ) : (
                        <div style={{ padding: 40, textAlign: 'center', color: '#999' }}>
                            <p>暂无运单数据</p>
                        </div>
                    )
                }}
            />

            <Modal
                title={editingShipment ? '编辑运单' : '创建运单'}
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={() => form.submit()}
                width="80%"
                style={{ maxWidth: 1200, minWidth: 800 }}
                centered
                styles={{ body: { maxHeight: '70vh', overflowY: 'auto', padding: '12px 20px' } }}
            >
                <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ origin_radius: 1000, dest_radius: 1000, auto_status_enabled: true, transport_mode: 'lcl', cargo_type: 'general' }} size="middle">

                    {/* ========== 第一区：路线与时效 ========== */}
                    <Divider orientation={"left" as any} plain className={`${styles.sectionDivider} ${styles.routeDivider}`}>📍 路线与时效</Divider>

                    {/* ETD/ETA + 发货地/目的地预览（路由摘要） */}
                    <Row gutter={12} align="top">
                        {/* 左侧：时间选择器 */}
                        <Col span={6}>
                            <Form.Item name="etd" label="ETD 预计出发" className={styles.formItem}>
                                <DatePicker showTime style={{ width: '100%' }} placeholder="预计出发" />
                            </Form.Item>
                        </Col>
                        <Col span={6}>
                            <Form.Item name="eta" label="ETA 预计到达" className={styles.formItem}>
                                <DatePicker showTime style={{ width: '100%' }} placeholder="预计到达" />
                            </Form.Item>
                        </Col>
                        {/* 右侧：路由摘要，实时展示发货地→目的地 */}
                        <Col span={5}>
                            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.origin !== cur.origin}>
                                {({ getFieldValue }) => {
                                    const origin = getFieldValue('origin');
                                    return (
                                        <div className={styles.formItem}>
                                            <div style={{ marginBottom: 4, color: 'rgba(0, 0, 0, 0.85)', fontSize: 14 }}>发货地</div>
                                            <div style={{
                                                height: 32,
                                                lineHeight: '32px',
                                                fontSize: 14,
                                                color: origin ? '#1890ff' : '#bfbfbf',
                                                padding: '0 11px',
                                                background: '#f0f5ff',
                                                border: '1px solid #d6e4ff',
                                                borderRadius: 6
                                            }}>
                                                {origin || '自动获取'}
                                            </div>
                                        </div>
                                    );
                                }}
                            </Form.Item>
                        </Col>
                        <Col span={1} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', paddingTop: 28 }}>
                            <span style={{ fontSize: 16, color: '#52c41a' }}>→</span>
                        </Col>
                        <Col span={5}>
                            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.destination !== cur.destination}>
                                {({ getFieldValue }) => {
                                    const destination = getFieldValue('destination');
                                    return (
                                        <div className={styles.formItem}>
                                            <div style={{ marginBottom: 4, color: 'rgba(0, 0, 0, 0.85)', fontSize: 14 }}>目的地</div>
                                            <div style={{
                                                height: 32,
                                                lineHeight: '32px',
                                                fontSize: 14,
                                                color: destination ? '#52c41a' : '#bfbfbf',
                                                padding: '0 11px',
                                                background: '#f6ffed',
                                                border: '1px solid #b7eb8f',
                                                borderRadius: 6
                                            }}>
                                                {destination || '自动获取'}
                                            </div>
                                        </div>
                                    );
                                }}
                            </Form.Item>
                        </Col>
                    </Row>

                    {/* 发货人/收货人 */}
                    <Row gutter={12} align="middle">
                        <Col span={10}>
                            <Row gutter={12}>
                                <Col span={12}>
                                    <Form.Item name="sender_name" label="发货人" rules={[{ required: true, message: '请输入发货人' }]} className={styles.formItem}>
                                        <Input placeholder="姓名" />
                                    </Form.Item>
                                </Col>
                                <Col span={12}>
                                    <Form.Item name="sender_phone" label="发货电话" rules={[{ required: true, message: '请输入发货电话' }]} className={styles.formItem}>
                                        <AutoComplete
                                            placeholder="输入电话自动搜索"
                                            options={senderOptions}
                                            onSearch={(val) => handleCustomerSearch(val, 'sender')}
                                            onSelect={(val, opt) => onCustomerSelect(val, opt, 'sender')}
                                        />
                                    </Form.Item>
                                </Col>
                            </Row>
                        </Col>
                        <Col span={2} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <span className={styles.arrowIconLarge}>→</span>
                        </Col>
                        <Col span={10}>
                            <Row gutter={12}>
                                <Col span={12}>
                                    <Form.Item name="receiver_name" label="收货人" rules={[{ required: true, message: '请输入收货人' }]} className={styles.formItem}>
                                        <Input placeholder="姓名" />
                                    </Form.Item>
                                </Col>
                                <Col span={12}>
                                    <Form.Item name="receiver_phone" label="收货电话" rules={[{ required: true, message: '请输入收货电话' }]} className={styles.formItem}>
                                        <AutoComplete
                                            placeholder="输入电话自动搜索"
                                            options={receiverOptions}
                                            onSearch={(val) => handleCustomerSearch(val, 'receiver')}
                                            onSelect={(val, opt) => onCustomerSelect(val, opt, 'receiver')}
                                        />
                                    </Form.Item>
                                </Col>
                            </Row>
                        </Col>
                    </Row>

                    {/* 发货地址 → 收货地址 */}
                    <Row gutter={12}>
                        <Col span={10}>
                            <Form.Item name="origin_address" label="发货地址" rules={[{ required: true, message: '请输入发货地址' }]} className={styles.formItem}>
                                <AddressInput
                                    placeholder="输入发货地址..."
                                    onAddressSelect={(data: AddressData) => {
                                        setOriginLocation({ lat: data.lat, lng: data.lng });
                                        form.setFieldsValue({
                                            origin: data.shortName,
                                            origin_address: data.address
                                        });
                                    }}
                                />
                            </Form.Item>
                            <Form.Item name="origin" hidden><Input /></Form.Item>
                        </Col>
                        <Col span={2} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', paddingTop: 24 }}>
                            <span style={{ fontSize: 18, color: '#52c41a' }}>→</span>
                        </Col>
                        <Col span={10}>
                            <Form.Item name="dest_address" label="收货地址" rules={[{ required: true, message: '请输入收货地址' }]} className={styles.formItem}>
                                <AddressInput
                                    placeholder="输入收货地址..."
                                    onAddressSelect={(data: AddressData) => {
                                        setDestLocation({ lat: data.lat, lng: data.lng });
                                        form.setFieldsValue({
                                            destination: data.shortName,
                                            dest_address: data.address
                                        });
                                    }}
                                />
                            </Form.Item>
                            <Form.Item name="destination" hidden><Input /></Form.Item>
                        </Col>
                    </Row>

                    {/* ========== 第二区：运输配置 ========== */}
                    <Divider orientation={"left" as any} plain className={`${styles.sectionDivider} ${styles.transportDivider}`}>🚚 运输配置</Divider>

                    <Row gutter={12}>
                        <Col span={4}>
                            <Form.Item name="transport_type" label="运输类型" rules={[{ required: true }]} className={styles.formItem}>
                                <Select placeholder="类型">
                                    <Select.Option value="sea">🚢 海运</Select.Option>
                                    <Select.Option value="air">✈️ 空运</Select.Option>
                                    <Select.Option value="land">🚚 陆运</Select.Option>
                                    <Select.Option value="multimodal">🔄 多式联运</Select.Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={4}>
                            <Form.Item name="transport_mode" label="运输模式" className={styles.formItem}>
                                <Select placeholder="模式">
                                    <Select.Option value="lcl">📦 零担</Select.Option>
                                    <Select.Option value="fcl">🚢 整柜</Select.Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Form.Item noStyle shouldUpdate={(prev, cur) => prev.transport_mode !== cur.transport_mode}>
                            {({ getFieldValue }) =>
                                getFieldValue('transport_mode') === 'fcl' ? (
                                    <Col span={4}>
                                        <Form.Item name="container_type" label="柜型" className={styles.formItem}>
                                            <Select placeholder="柜型" allowClear>
                                                <Select.Option value="20GP">20GP</Select.Option>
                                                <Select.Option value="40GP">40GP</Select.Option>
                                                <Select.Option value="40HQ">40HQ</Select.Option>
                                                <Select.Option value="45HQ">45HQ</Select.Option>
                                            </Select>
                                        </Form.Item>
                                    </Col>
                                ) : null
                            }
                        </Form.Item>
                        <Col span={4}>
                            <Form.Item name="cargo_type" label="货物类型" className={styles.formItem}>
                                <Select placeholder="类型">
                                    <Select.Option value="general">普货</Select.Option>
                                    <Select.Option value="dangerous">危险品</Select.Option>
                                    <Select.Option value="cold_chain">冷链</Select.Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={6}>
                            <Form.Item name="device_id" label="定位设备ID号" className={styles.formItem}>
                                <Select
                                    placeholder="输入设备ID搜索"
                                    allowClear
                                    showSearch
                                    filterOption={(input, option) =>
                                        (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
                                    }
                                    optionFilterProp="children"
                                >
                                    {devices.map((d) => (
                                        <Select.Option key={d.id} value={d.id}>
                                            {d.external_device_id || d.id}
                                        </Select.Option>
                                    ))}
                                </Select>
                            </Form.Item>
                        </Col>
                    </Row>

                    {/* ========== 第三区：货物与单证 ========== */}
                    <Divider orientation={"left" as any} plain className={`${styles.sectionDivider} ${styles.cargoDivider}`}>📦 货物与单证</Divider>

                    {/* 货物信息 */}
                    <Row gutter={12}>
                        <Col span={6}>
                            <Form.Item name="cargo_name" label="货物名称" className={styles.formItem}>
                                <Input placeholder="货物描述" />
                            </Form.Item>
                        </Col>
                        <Col span={4}>
                            <Form.Item name="pieces" label="件数" className={styles.formItem}>
                                <InputNumber min={0} style={{ width: '100%' }} placeholder="件" />
                            </Form.Item>
                        </Col>
                        <Col span={4}>
                            <Form.Item name="weight" label="重量(kg)" className={styles.formItem}>
                                <InputNumber min={0} step={0.1} style={{ width: '100%' }} placeholder="kg" />
                            </Form.Item>
                        </Col>
                        <Col span={4}>
                            <Form.Item name="volume" label="体积(m³)" className={styles.formItem}>
                                <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="m³" />
                            </Form.Item>
                        </Col>
                        {isFieldVisible('bill_of_lading') && (
                            <Col span={6}>
                                <Form.Item name="bill_of_lading" label="提单号" className={styles.formItem}>
                                    <Input placeholder="MBL/HBL/AWB" />
                                </Form.Item>
                            </Col>
                        )}
                    </Row>

                    {/* 单证船务 */}
                    <Row gutter={12}>
                        {isFieldVisible('container_no') && (
                            <Col span={4}>
                                <Form.Item name="container_no" label="箱号/车牌" className={styles.formItem}>
                                    <Input placeholder="MSKU1234567" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('seal_no') && (
                            <Col span={4}>
                                <Form.Item name="seal_no" label="封条号" className={styles.formItem}>
                                    <Input placeholder="Seal Number" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('vessel_name') && (
                            <Col span={4}>
                                <Form.Item name="vessel_name" label="船名" className={styles.formItem}>
                                    <Input placeholder="如: EVER GIVEN" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('voyage_no') && (
                            <Col span={4}>
                                <Form.Item name="voyage_no" label="航次" className={styles.formItem}>
                                    <Input placeholder="如: 2026W02" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('carrier') && (
                            <Col span={4}>
                                <Form.Item name="carrier" label="船司/航司" className={styles.formItem}>
                                    <Input placeholder="如: COSCO" />
                                </Form.Item>
                            </Col>
                        )}
                    </Row>

                    {/* 订单关联 (归类到货物与单证) */}
                    <Row gutter={12}>
                        {isFieldVisible('po_numbers') && (
                            <Col span={8}>
                                <Form.Item name="po_numbers" label="PO单号" className={styles.formItem}>
                                    <Input placeholder="多个用逗号分隔" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('sku_ids') && (
                            <Col span={8}>
                                <Form.Item name="sku_ids" label="SKU ID" className={styles.formItem}>
                                    <Input placeholder="多个用逗号分隔" />
                                </Form.Item>
                            </Col>
                        )}
                        {isFieldVisible('fba_shipment_id') && (
                            <Col span={8}>
                                <Form.Item name="fba_shipment_id" label="FBA发货编号" className={styles.formItem}>
                                    <Input placeholder="如: FBA1234567" />
                                </Form.Item>
                            </Col>
                        )}
                    </Row>

                    {/* ========== 第四区：费用信息 ========== */}
                    <Divider orientation={"left" as any} plain className={`${styles.sectionDivider} ${styles.costDivider}`}>💰 费用信息</Divider>

                    {/* 计算费用字段总数：运费(1) + 可见附加费 + 总费用(1) */}
                    {(() => {
                        const visibleFeeCount = 1 + (isFieldVisible('surcharges') ? 1 : 0) + (isFieldVisible('customs_fee') ? 1 : 0) + (isFieldVisible('other_cost') ? 1 : 0);
                        const feeColSpan = Math.floor(24 / (visibleFeeCount + 1)); // +1 是总费用
                        return (
                            <Row gutter={12}>
                                {/* 运费始终显示 */}
                                <Col span={feeColSpan}>
                                    <Form.Item name="freight_cost" label={`运费(${currencyConfig.code})`} className={styles.formItem}>
                                        <InputNumber
                                            min={0}
                                            step={0.01}
                                            style={{ width: '100%' }}
                                            placeholder="0.00"
                                            onChange={() => {
                                                const values = form.getFieldsValue(['freight_cost', 'surcharges', 'customs_fee', 'other_cost']);
                                                let total = values.freight_cost || 0;
                                                if (isFieldVisible('surcharges')) total += values.surcharges || 0;
                                                if (isFieldVisible('customs_fee')) total += values.customs_fee || 0;
                                                if (isFieldVisible('other_cost')) total += values.other_cost || 0;
                                                form.setFieldsValue({ total_cost: total });
                                            }}
                                        />
                                    </Form.Item>
                                </Col>
                                {isFieldVisible('surcharges') && (
                                    <Col span={feeColSpan}>
                                        <Form.Item name="surcharges" label={`附加费(${currencyConfig.code})`} className={styles.formItem}>
                                            <InputNumber
                                                min={0}
                                                step={0.01}
                                                style={{ width: '100%' }}
                                                placeholder="0.00"
                                                onChange={() => {
                                                    const values = form.getFieldsValue(['freight_cost', 'surcharges', 'customs_fee', 'other_cost']);
                                                    let total = values.freight_cost || 0;
                                                    if (isFieldVisible('surcharges')) total += values.surcharges || 0;
                                                    if (isFieldVisible('customs_fee')) total += values.customs_fee || 0;
                                                    if (isFieldVisible('other_cost')) total += values.other_cost || 0;
                                                    form.setFieldsValue({ total_cost: total });
                                                }}
                                            />
                                        </Form.Item>
                                    </Col>
                                )}
                                {isFieldVisible('customs_fee') && (
                                    <Col span={feeColSpan}>
                                        <Form.Item name="customs_fee" label={`关税(${currencyConfig.code})`} className={styles.formItem}>
                                            <InputNumber
                                                min={0}
                                                step={0.01}
                                                style={{ width: '100%' }}
                                                placeholder="0.00"
                                                onChange={() => {
                                                    const values = form.getFieldsValue(['freight_cost', 'surcharges', 'customs_fee', 'other_cost']);
                                                    let total = values.freight_cost || 0;
                                                    if (isFieldVisible('surcharges')) total += values.surcharges || 0;
                                                    if (isFieldVisible('customs_fee')) total += values.customs_fee || 0;
                                                    if (isFieldVisible('other_cost')) total += values.other_cost || 0;
                                                    form.setFieldsValue({ total_cost: total });
                                                }}
                                            />
                                        </Form.Item>
                                    </Col>
                                )}
                                {isFieldVisible('other_cost') && (
                                    <Col span={feeColSpan}>
                                        <Form.Item name="other_cost" label={`其他(${currencyConfig.code})`} className={styles.formItem}>
                                            <InputNumber
                                                min={0}
                                                step={0.01}
                                                style={{ width: '100%' }}
                                                placeholder="0.00"
                                                onChange={() => {
                                                    const values = form.getFieldsValue(['freight_cost', 'surcharges', 'customs_fee', 'other_cost']);
                                                    let total = values.freight_cost || 0;
                                                    if (isFieldVisible('surcharges')) total += values.surcharges || 0;
                                                    if (isFieldVisible('customs_fee')) total += values.customs_fee || 0;
                                                    if (isFieldVisible('other_cost')) total += values.other_cost || 0;
                                                    form.setFieldsValue({ total_cost: total });
                                                }}
                                            />
                                        </Form.Item>
                                    </Col>
                                )}
                                {/* 总费用始终显示，使用与其他费用字段相同的动态 span */}
                                <Col span={feeColSpan}>
                                    {/* 总费用 = 运费 + 附加费 + 关税 + 其他费用（仅包含已开启的字段），自动计算，只读 */}
                                    <Form.Item
                                        noStyle
                                        shouldUpdate={(prev, cur) => {
                                            if (prev.freight_cost !== cur.freight_cost) return true;
                                            if (isFieldVisible('surcharges') && prev.surcharges !== cur.surcharges) return true;
                                            if (isFieldVisible('customs_fee') && prev.customs_fee !== cur.customs_fee) return true;
                                            if (isFieldVisible('other_cost') && prev.other_cost !== cur.other_cost) return true;
                                            return false;
                                        }}
                                    >
                                        {({ getFieldValue }) => {
                                            let total = getFieldValue('freight_cost') || 0;
                                            if (isFieldVisible('surcharges')) total += getFieldValue('surcharges') || 0;
                                            if (isFieldVisible('customs_fee')) total += getFieldValue('customs_fee') || 0;
                                            if (isFieldVisible('other_cost')) total += getFieldValue('other_cost') || 0;
                                            return (
                                                <div className={styles.formItem}>
                                                    <div style={{ marginBottom: 4, color: 'rgba(0, 0, 0, 0.85)', fontSize: 14 }}>
                                                        总费用({currencyConfig.code})
                                                    </div>
                                                    <div style={{
                                                        height: 32,
                                                        lineHeight: '32px',
                                                        fontSize: 16,
                                                        fontWeight: 600,
                                                        color: total > 0 ? '#faad14' : '#bfbfbf',
                                                        padding: '0 11px',
                                                        background: '#fffbe6',
                                                        border: '1px solid #ffe58f',
                                                        borderRadius: 6
                                                    }}>
                                                        {currencyConfig.symbol}{total.toFixed(2)}
                                                    </div>
                                                    <Form.Item name="total_cost" hidden><InputNumber /></Form.Item>
                                                </div>
                                            );
                                        }}
                                    </Form.Item>
                                </Col>
                            </Row>
                        );
                    })()}

                </Form>
            </Modal>

            {/* 运单详情弹窗 - 方案A：分区卡片式布局 */}
            <Modal
                title={<span><EyeOutlined style={{ marginRight: 8 }} />运单详情 - {viewingShipment?.id}</span>}
                open={detailModalVisible}
                onCancel={() => setDetailModalVisible(false)}
                footer={[
                    <Button key="close" onClick={() => setDetailModalVisible(false)}>关闭</Button>,
                    canEdit && <Button key="edit" type="primary" onClick={() => {
                        setDetailModalVisible(false);
                        if (viewingShipment) handleEdit(viewingShipment);
                    }}>编辑运单</Button>
                ]}
                width={1100}
            >
                {viewingShipment && (
                    <Tabs
                        activeKey={detailActiveTab}
                        onChange={setDetailActiveTab}
                        items={[
                            {
                                key: 'info',
                                label: '基本信息',
                                children: (
                                    <div style={{ maxHeight: 500, overflowY: 'auto', paddingRight: 8 }}>
                                        {/* ===== 第一区：路线与时效 ===== */}
                                        <Card
                                            size="small"
                                            title={<span style={{ color: '#52c41a' }}>📍 路线与时效</span>}
                                            style={{ marginBottom: 16 }}
                                        >
                                            <Descriptions column={3} size="small" bordered labelStyle={{ whiteSpace: 'nowrap' }}>
                                                <Descriptions.Item label="运单号">{viewingShipment.id}</Descriptions.Item>
                                                <Descriptions.Item label="状态">
                                                    <Tag color={statusColors[viewingShipment.status]}>
                                                        {statusLabels[viewingShipment.status] || viewingShipment.status}
                                                    </Tag>
                                                </Descriptions.Item>
                                                <Descriptions.Item label="进度">
                                                    <Progress percent={viewingShipment.progress || 0} size="small" style={{ width: 100 }} />
                                                </Descriptions.Item>
                                                <Descriptions.Item label="发货地">{viewingShipment.origin || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="目的地">{viewingShipment.destination || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="组织机构">{viewingShipment.org_name || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="发货人">{viewingShipment.sender_name || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="发货电话" span={2}>{viewingShipment.sender_phone || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="发货地址" span={3}>{viewingShipment.origin_address || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="收货人">{viewingShipment.receiver_name || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="收货电话" span={2}>{viewingShipment.receiver_phone || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="收货地址" span={3}>{viewingShipment.dest_address || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="ETD 预计出发">
                                                    {viewingShipment.etd ? new Date(viewingShipment.etd).toLocaleString('zh-CN') : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="ETA 预计到达">
                                                    {viewingShipment.eta ? new Date(viewingShipment.eta).toLocaleString('zh-CN') : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="创建时间">
                                                    {viewingShipment.created_at ? new Date(viewingShipment.created_at).toLocaleString('zh-CN') : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="ATD 实际出发">
                                                    {viewingShipment.atd ? new Date(viewingShipment.atd).toLocaleString('zh-CN') : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="ATA 实际到达">
                                                    {viewingShipment.ata ? new Date(viewingShipment.ata).toLocaleString('zh-CN') : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="定位设备ID号">
                                                    {(() => {
                                                        const device = viewingShipment.device;
                                                        if (device?.external_device_id) return device.external_device_id;
                                                        if (viewingShipment.device_id) {
                                                            const found = devices.find(d => d.id === viewingShipment.device_id);
                                                            return found?.external_device_id || viewingShipment.device_id;
                                                        }
                                                        if (viewingShipment.unbound_device_id) {
                                                            const found = devices.find(d => d.id === viewingShipment.unbound_device_id);
                                                            return found?.external_device_id || viewingShipment.unbound_device_id;
                                                        }
                                                        return <Tag color="orange">未绑定</Tag>;
                                                    })()}
                                                </Descriptions.Item>
                                            </Descriptions>
                                        </Card>

                                        {/* ===== 第二区：运输配置 ===== */}
                                        <Card
                                            size="small"
                                            title={<span style={{ color: '#1890ff' }}>🚚 运输配置</span>}
                                            style={{ marginBottom: 16 }}
                                        >
                                            <Descriptions column={3} size="small" bordered>
                                                <Descriptions.Item label="运输类型">
                                                    {viewingShipment.transport_type === 'sea' ? '海运' :
                                                        viewingShipment.transport_type === 'air' ? '空运' :
                                                            viewingShipment.transport_type === 'land' ? '陆运' :
                                                                viewingShipment.transport_type === 'multimodal' ? '多式联运' : viewingShipment.transport_type || '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="货物类型">
                                                    {viewingShipment.cargo_type === 'general' ? '普货' :
                                                        viewingShipment.cargo_type === 'dangerous' ? '危险品' :
                                                            viewingShipment.cargo_type === 'cold_chain' ? '冷链' : viewingShipment.cargo_type || '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="运输模式">
                                                    {viewingShipment.transport_mode === 'lcl' ? '零担' :
                                                        viewingShipment.transport_mode === 'fcl' ? '整柜' : viewingShipment.transport_mode || '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="柜型">{viewingShipment.container_type || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="提单号">{viewingShipment.bill_of_lading || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="箱号/车牌">{viewingShipment.container_no || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="船名">{viewingShipment.vessel_name || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="航次">{viewingShipment.voyage_no || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="船司/航司">{viewingShipment.carrier || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="封条号">{viewingShipment.seal_no || '-'}</Descriptions.Item>
                                            </Descriptions>
                                        </Card>

                                        {/* ===== 第三区：货物与单证 ===== */}
                                        <Card
                                            size="small"
                                            title={<span style={{ color: '#722ed1' }}>📦 货物与单证</span>}
                                            style={{ marginBottom: 16 }}
                                        >
                                            <Descriptions column={3} size="small" bordered>
                                                <Descriptions.Item label="货物名称">{viewingShipment.cargo_name || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="件数">
                                                    {viewingShipment.pieces != null ? viewingShipment.pieces.toLocaleString() : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="重量(kg)">
                                                    {viewingShipment.weight != null ? viewingShipment.weight.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="体积(m³)">
                                                    {viewingShipment.volume != null ? viewingShipment.volume.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) : '-'}
                                                </Descriptions.Item>
                                                <Descriptions.Item label="PO单号">{viewingShipment.po_numbers || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="SKU ID">{viewingShipment.sku_ids || '-'}</Descriptions.Item>
                                                <Descriptions.Item label="FBA编号" span={3}>{viewingShipment.fba_shipment_id || '-'}</Descriptions.Item>
                                            </Descriptions>
                                        </Card>

                                        {/* ===== 第四区：费用信息 ===== */}
                                        <Card
                                            size="small"
                                            title={<span style={{ color: '#faad14' }}>💰 费用信息</span>}
                                        >
                                            <Descriptions column={3} size="small" bordered>
                                                <Descriptions.Item label={`运费(${currencyConfig.code})`}>
                                                    {formatAmount(viewingShipment.freight_cost)}
                                                </Descriptions.Item>
                                                <Descriptions.Item label={`附加费(${currencyConfig.code})`}>
                                                    {formatAmount(viewingShipment.surcharges)}
                                                </Descriptions.Item>
                                                <Descriptions.Item label={`关税(${currencyConfig.code})`}>
                                                    {formatAmount(viewingShipment.customs_fee)}
                                                </Descriptions.Item>
                                                <Descriptions.Item label={`其他费用(${currencyConfig.code})`}>
                                                    {formatAmount(viewingShipment.other_cost)}
                                                </Descriptions.Item>
                                                <Descriptions.Item label={`总费用(${currencyConfig.code})`} span={2}>
                                                    <span style={{ color: '#faad14', fontWeight: 600, fontSize: 16 }}>
                                                        {formatAmount(viewingShipment.total_cost)}
                                                    </span>
                                                </Descriptions.Item>
                                            </Descriptions>
                                        </Card>
                                    </div>
                                ),
                            },
                            {
                                key: 'logs',
                                label: '操作日志',
                                children: <ShipmentLogTimeline shipmentId={viewingShipment.id} visible={true} />,
                            },
                        ]}
                    />
                )}
            </Modal>

            {/* 运输环节管理弹窗 */}
            <TransportStageModal
                shipment={stageModalShipment}
                visible={!!stageModalShipment}
                onClose={() => setStageModalShipment(null)}
                onStageChange={() => loadData()}
            />
        </Card>
    );
};

export default Shipments;
