import axios from 'axios';
import type { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import type { ApiResponse, LoginRequest, LoginResponse, User, Device, Shipment, Alert, DashboardStats, TrackPoint, ShipmentLog, DeviceBinding, CarrierTrack, ShipmentMilestone, Partner, PartnerCreateRequest, PartnerUpdateRequest, ShipmentCollaboration, CollaborationCreateRequest, FreightRate, RateCreateRequest, RateCompareRequest, RateCompareResult, RouteInfo, PartnerPerformance, ShipmentDocument, GeneratedDocument, OCRResult, DocumentTemplate, DocumentStatus, Customer, PortGeofence, SendSMSCodeRequest, SMSLoginRequest, SMSLoginResponse, UserOrg } from '../types';

const API_BASE_URL = '/api';
console.log('🔌 API_BASE_URL forced to:', API_BASE_URL);


class ApiClient {
    private client: AxiosInstance;

    constructor() {
        this.client = axios.create({
            baseURL: API_BASE_URL,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json',
            },
        });

        // 请求拦截器
        this.client.interceptors.request.use(
            (config: InternalAxiosRequestConfig) => {
                const token = localStorage.getItem('token');
                if (token && config.headers) {
                    config.headers.Authorization = `Bearer ${token}`;
                }

                // 自动添加当前选中的组织ID到请求参数
                // 注意：只有当请求params中没有org_id属性时才自动添加
                // 如果请求已有org_id（包括空字符串），说明调用方明确指定了筛选条件
                const currentOrgId = localStorage.getItem('currentOrgId');
                if (currentOrgId && config.params !== undefined) {
                    // 检查是否已有 org_id 属性（使用 in 判断，允许空值）
                    if (!('org_id' in config.params)) {
                        config.params = { ...config.params, org_id: currentOrgId };
                    }
                }

                return config;
            },
            (error: AxiosError) => Promise.reject(error)
        );

        // 响应拦截器
        this.client.interceptors.response.use(
            (response: AxiosResponse) => response,
            async (error: AxiosError) => {
                if (error.response?.status === 401) {
                    localStorage.removeItem('token');
                    localStorage.removeItem('user');
                    window.location.href = '/login';
                }

                // 处理可能由无效 org_id 导致的 500 错误
                // 如果请求包含 org_id 参数且返回 500，尝试清除并重试一次
                if (error.response?.status === 500 && error.config) {
                    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };
                    const params = originalRequest.params as Record<string, unknown> | undefined;

                    if (params?.org_id && !originalRequest._retry) {
                        console.warn('API 500 error with org_id, clearing and retrying:', params.org_id);
                        localStorage.removeItem('currentOrgId');
                        originalRequest._retry = true;
                        // 移除 org_id 参数重试
                        delete params.org_id;
                        return this.client.request(originalRequest);
                    }
                }

                return Promise.reject(error);
            }
        );
    }

    // 认证
    async login(data: LoginRequest): Promise<LoginResponse> {
        const response = await this.client.post<LoginResponse>('/auth/login', data);
        return response.data;
    }



    async sendSMSCode(data: SendSMSCodeRequest): Promise<ApiResponse<{ cooldown_seconds: number; debug_code?: string }>> {
        const response = await this.client.post<ApiResponse<{ cooldown_seconds: number; debug_code?: string }>>('/auth/sms/send-code', data);
        return response.data;
    }

    async loginBySMS(data: SMSLoginRequest): Promise<ApiResponse<SMSLoginResponse>> {
        const response = await this.client.post<ApiResponse<SMSLoginResponse>>('/auth/sms/login', data);
        return response.data;
    }

    async selectOrg(org_id: string): Promise<ApiResponse<{ token: string; current_org: UserOrg }>> {
        const response = await this.client.post<ApiResponse<{ token: string; current_org: UserOrg }>>('/auth/select-org', { org_id });
        return response.data;
    }

    async resetPasswordBySMS(data: { phone_country_code?: string; phone_number: string; code: string; new_password: string }): Promise<ApiResponse<void>> {
        const response = await this.client.post<ApiResponse<void>>('/auth/password/reset-by-sms', data);
        return response.data;
    }

    async getCurrentUser(): Promise<ApiResponse<User>> {
        const response = await this.client.get<ApiResponse<User>>('/auth/me');
        return response.data;
    }

    async changePassword(data: { current_password: string; new_password: string }): Promise<ApiResponse<void>> {
        const response = await this.client.post<ApiResponse<void>>('/auth/change-password', data);
        return response.data;
    }

    // 设备
    async getDevices(params?: { status?: string; type?: string; search?: string; syncExternal?: boolean; org_id?: string }): Promise<ApiResponse<Device[]>> {
        const response = await this.client.get<ApiResponse<Device[]>>('/devices', { params });
        return response.data;
    }

    async getDevice(id: string): Promise<ApiResponse<Device>> {
        const response = await this.client.get<ApiResponse<Device>>(`/devices/${id}`);
        return response.data;
    }

    async createDevice(data: Partial<Device>): Promise<ApiResponse<Device>> {
        const response = await this.client.post<ApiResponse<Device>>('/devices', data);
        return response.data;
    }

    async updateDevice(id: string, data: Partial<Device>): Promise<ApiResponse<Device>> {
        const response = await this.client.put<ApiResponse<Device>>(`/devices/${id}`, data);
        return response.data;
    }

    async deleteDevice(id: string): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/devices/${id}`);
        return response.data;
    }

    async getDeviceTrack(id: string, startTime?: string, endTime?: string): Promise<ApiResponse<TrackPoint[]>> {
        const response = await this.client.get<ApiResponse<TrackPoint[]>>(`/devices/${id}/track`, {
            params: { startTime, endTime },
            timeout: 60000 // 增加超时时间到60秒，以支持长周期轨迹查询
        });
        return response.data;
    }

    // 运单
    async getShipments(params?: { status?: string; search?: string; org_id?: string; limit?: number }): Promise<ApiResponse<Shipment[]>> {
        const response = await this.client.get<ApiResponse<Shipment[]>>('/shipments', { params });
        return response.data;
    }

    async getShipment(id: string): Promise<ApiResponse<Shipment>> {
        const response = await this.client.get<ApiResponse<Shipment>>(`/shipments/${id}`);
        return response.data;
    }

    async createShipment(data: Partial<Shipment>): Promise<ApiResponse<Shipment>> {
        const response = await this.client.post<ApiResponse<Shipment>>('/shipments', data);
        return response.data;
    }

    async updateShipment(id: string, data: Partial<Shipment>): Promise<ApiResponse<Shipment>> {
        const response = await this.client.put<ApiResponse<Shipment>>(`/shipments/${id}`, data);
        return response.data;
    }

    async deleteShipment(id: string): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/shipments/${id}`);
        return response.data;
    }

    async getShipmentRoute(id: string): Promise<ApiResponse<TrackPoint[]>> {
        const response = await this.client.get<ApiResponse<TrackPoint[]>>(`/shipments/${id}/route`);
        return response.data;
    }

    // 快捷切换运单状态
    async transitionShipmentStatus(id: string, action: 'depart' | 'deliver' | 'cancel', receiver?: string): Promise<ApiResponse<{ shipment: Shipment; message: string }>> {
        const response = await this.client.post<ApiResponse<{ shipment: Shipment; message: string }>>(`/shipments/${id}/transition`, { action, receiver });
        return response.data;
    }

    async getShipmentLogs(id: string): Promise<ApiResponse<ShipmentLog[]>> {
        const response = await this.client.get<ApiResponse<ShipmentLog[]>>(`/shipments/${id}/logs`);
        return response.data;
    }

    // 获取运单轨迹数据（包含设备当前位置）
    async getShipmentTracks(id: string): Promise<ApiResponse<{
        device_id: string;
        shipment_id: string;
        tracks: Array<{
            id: number;
            device_id: string;
            lat: number;
            lng: number;
            speed: number;
            direction: number;
            temperature?: number;
            humidity?: number;
            locate_time: string;
        }>;
        current_position: {
            lat: number;
            lng: number;
            speed: number;
            timestamp: string;
        } | null;
    }>> {
        const response = await this.client.get<ApiResponse<{
            device_id: string;
            shipment_id: string;
            tracks: Array<{
                id: number;
                device_id: string;
                lat: number;
                lng: number;
                speed: number;
                direction: number;
                temperature?: number;
                humidity?: number;
                locate_time: string;
            }>;
            current_position: {
                lat: number;
                lng: number;
                speed: number;
                timestamp: string;
            } | null;
        }>>(`/shipments/${id}/tracks`);
        return response.data;
    }

    async getShipmentBindings(id: string): Promise<ApiResponse<DeviceBinding[]>> {
        const response = await this.client.get<ApiResponse<DeviceBinding[]>>(`/shipments/${id}/bindings`);
        return response.data;
    }

    // Phase 2: 船司追踪
    async getCarrierTracks(shipmentId: string): Promise<ApiResponse<CarrierTrack[]>> {
        const response = await this.client.get<ApiResponse<CarrierTrack[]>>(`/shipments/${shipmentId}/carrier-tracks`);
        return response.data;
    }

    async getMilestones(shipmentId: string): Promise<ApiResponse<ShipmentMilestone[]>> {
        const response = await this.client.get<ApiResponse<ShipmentMilestone[]>>(`/shipments/${shipmentId}/milestones`);
        return response.data;
    }

    async syncCarrierTrack(shipmentId: string): Promise<ApiResponse<{ synced_events: number; message: string }>> {
        const response = await this.client.post<ApiResponse<{ synced_events: number; message: string }>>(`/shipments/${shipmentId}/sync-carrier`);
        return response.data;
    }

    // Phase 6: 运输环节管理
    async getShipmentStages(shipmentId: string): Promise<ApiResponse<{ stages: any[]; current_stage: string; total_cost: number }>> {
        const response = await this.client.get<ApiResponse<{ stages: any[]; current_stage: string; total_cost: number }>>(`/shipments/${shipmentId}/stages`);
        return response.data;
    }

    async getShipmentStage(shipmentId: string, stageCode: string): Promise<ApiResponse<any>> {
        const response = await this.client.get<ApiResponse<any>>(`/shipments/${shipmentId}/stages/${stageCode}`);
        return response.data;
    }

    async updateShipmentStage(shipmentId: string, stageCode: string, data: any): Promise<ApiResponse<{ message: string; stage: any }>> {
        const response = await this.client.put<ApiResponse<{ message: string; stage: any }>>(`/shipments/${shipmentId}/stages/${stageCode}`, data);
        return response.data;
    }

    async transitionShipmentStage(shipmentId: string, note?: string): Promise<ApiResponse<{ message: string; current_stage: string; stages: any[] }>> {
        const response = await this.client.post<ApiResponse<{ message: string; current_stage: string; stages: any[] }>>(`/shipments/${shipmentId}/stages/transition`, { note });
        return response.data;
    }

    async startShipmentStage(shipmentId: string, stageCode: string, data?: { partner_id?: string; partner_name?: string; note?: string }): Promise<ApiResponse<{ message: string }>> {
        const response = await this.client.post<ApiResponse<{ message: string }>>(`/shipments/${shipmentId}/stages/${stageCode}/start`, data);
        return response.data;
    }

    async completeShipmentStage(shipmentId: string, stageCode: string, note?: string): Promise<ApiResponse<{ message: string }>> {
        const response = await this.client.post<ApiResponse<{ message: string }>>(`/shipments/${shipmentId}/stages/${stageCode}/complete`, { note });
        return response.data;
    }

    // 获取运单的设备停留记录
    async getShipmentStops(shipmentId: string, page?: number, pageSize?: number): Promise<ApiResponse<{
        records: Array<{
            id: string;
            device_id: string;
            device_external_id: string;
            shipment_id: string;
            start_time: string;
            end_time: string | null;
            duration_seconds: number;
            duration_text: string;
            latitude: number | null;
            longitude: number | null;
            address: string;
            status: 'active' | 'completed';
            alert_sent: boolean;
            created_at: string;
        }>;
        total: number;
        page: number;
        page_size: number;
    }>> {
        const response = await this.client.get<ApiResponse<{
            records: any[];
            total: number;
            page: number;
            page_size: number;
        }>>(`/shipments/${shipmentId}/stops`, { params: { page, page_size: pageSize } });
        return response.data;
    }

    // 获取运单途经城市
    async getTransitCities(shipmentId: string, refresh?: boolean): Promise<ApiResponse<Array<{
        id: string;
        country: string;
        province: string;
        city: string;
        latitude: number;
        longitude: number;
        entered_at: string;
        is_oversea: boolean;
    }>>> {
        const response = await this.client.get<ApiResponse<Array<{
            id: string;
            country: string;
            province: string;
            city: string;
            latitude: number;
            longitude: number;
            entered_at: string;
            is_oversea: boolean;
        }>>>(`/shipments/${shipmentId}/transit-cities`, { params: { refresh: refresh ? 'true' : undefined } });
        return response.data;
    }

    // 获取设备当前停留记录
    async getDeviceCurrentStop(deviceExternalId: string): Promise<ApiResponse<{
        id: string;
        device_id: string;
        device_external_id: string;
        shipment_id: string;
        start_time: string;
        end_time: string | null;
        duration_seconds: number;
        duration_text: string;
        latitude: number | null;
        longitude: number | null;
        address: string;
        status: 'active' | 'completed';
        alert_sent: boolean;
        created_at: string;
    }>> {
        const response = await this.client.get<ApiResponse<any>>(`/device-stops/current/${deviceExternalId}`);
        return response.data;
    }

    // Phase 6: 港口围栏
    async getPortGeofences(): Promise<ApiResponse<PortGeofence[]>> {
        const response = await this.client.get<ApiResponse<PortGeofence[]>>('/port-geofences');
        return response.data;
    }

    // Phase 3: 合作伙伴
    async getPartners(params?: { type?: string; status?: string; search?: string }): Promise<ApiResponse<Partner[]>> {
        const response = await this.client.get<ApiResponse<Partner[]>>('/partners', { params });
        return response.data;
    }

    async getPartner(id: string): Promise<ApiResponse<Partner>> {
        const response = await this.client.get<ApiResponse<Partner>>(`/partners/${id}`);
        return response.data;
    }

    async createPartner(data: PartnerCreateRequest): Promise<ApiResponse<Partner>> {
        const response = await this.client.post<ApiResponse<Partner>>('/partners', data);
        return response.data;
    }

    async updatePartner(id: string, data: PartnerUpdateRequest): Promise<ApiResponse<Partner>> {
        const response = await this.client.put<ApiResponse<Partner>>(`/partners/${id}`, data);
        return response.data;
    }

    async deletePartner(id: string): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/partners/${id}`);
        return response.data;
    }

    async getPartnerStats(id: string): Promise<ApiResponse<{ partner_id: string; total_collabs: number; completed_collabs: number; completion_rate: number; avg_duration_hrs: number }>> {
        const response = await this.client.get<ApiResponse<{ partner_id: string; total_collabs: number; completed_collabs: number; completion_rate: number; avg_duration_hrs: number }>>(`/partners/${id}/stats`);
        return response.data;
    }

    // Phase 3: 运单协作
    async getShipmentCollaborations(shipmentId: string): Promise<ApiResponse<ShipmentCollaboration[]>> {
        const response = await this.client.get<ApiResponse<ShipmentCollaboration[]>>(`/shipments/${shipmentId}/collaborations`);
        return response.data;
    }

    async assignPartnerToShipment(shipmentId: string, data: CollaborationCreateRequest): Promise<ApiResponse<ShipmentCollaboration>> {
        const response = await this.client.post<ApiResponse<ShipmentCollaboration>>(`/shipments/${shipmentId}/collaborations`, data);
        return response.data;
    }

    async updateCollaboration(id: number, data: { status?: string; remarks?: string }): Promise<ApiResponse<ShipmentCollaboration>> {
        const response = await this.client.put<ApiResponse<ShipmentCollaboration>>(`/collaborations/${id}`, data);
        return response.data;
    }

    // Phase 4: 运价管理
    async getRates(params?: { origin?: string; destination?: string; partner_id?: string; container_type?: string; active?: string }): Promise<ApiResponse<FreightRate[]>> {
        const response = await this.client.get<ApiResponse<FreightRate[]>>('/rates', { params });
        return response.data;
    }

    async getRate(id: number): Promise<ApiResponse<FreightRate>> {
        const response = await this.client.get<ApiResponse<FreightRate>>(`/rates/${id}`);
        return response.data;
    }

    async createRate(data: RateCreateRequest): Promise<ApiResponse<FreightRate>> {
        const response = await this.client.post<ApiResponse<FreightRate>>('/rates', data);
        return response.data;
    }

    async updateRate(id: number, data: Partial<RateCreateRequest>): Promise<ApiResponse<FreightRate>> {
        const response = await this.client.put<ApiResponse<FreightRate>>(`/rates/${id}`, data);
        return response.data;
    }

    async deleteRate(id: number): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/rates/${id}`);
        return response.data;
    }

    async compareRates(data: RateCompareRequest): Promise<ApiResponse<{ origin: string; destination: string; container_type: string; total_options: number; lowest_price: number; rates: RateCompareResult[] }>> {
        const response = await this.client.post<ApiResponse<{ origin: string; destination: string; container_type: string; total_options: number; lowest_price: number; rates: RateCompareResult[] }>>('/rates/compare', data);
        return response.data;
    }

    async getAvailableRoutes(): Promise<ApiResponse<RouteInfo[]>> {
        const response = await this.client.get<ApiResponse<RouteInfo[]>>('/rates/routes');
        return response.data;
    }

    async getPartnerPerformance(params?: { partner_id?: string; route_lane?: string }): Promise<ApiResponse<PartnerPerformance[]>> {
        const response = await this.client.get<ApiResponse<PartnerPerformance[]>>('/rates/performance', { params });
        return response.data;
    }

    // Phase 5: 文档管理
    async getShipmentDocuments(shipmentId: string): Promise<ApiResponse<ShipmentDocument[]>> {
        const response = await this.client.get<ApiResponse<ShipmentDocument[]>>(`/shipments/${shipmentId}/documents`);
        return response.data;
    }

    async uploadDocument(shipmentId: string, formData: FormData): Promise<ApiResponse<ShipmentDocument>> {
        const response = await this.client.post<ApiResponse<ShipmentDocument>>(`/shipments/${shipmentId}/documents`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
        });
        return response.data;
    }

    async downloadDocument(docId: number): Promise<Blob> {
        const response = await this.client.get(`/documents/${docId}/download`, { responseType: 'blob' });
        return response.data;
    }

    async reviewDocument(docId: number, data: { status: DocumentStatus; remarks?: string }): Promise<ApiResponse<ShipmentDocument>> {
        const response = await this.client.put<ApiResponse<ShipmentDocument>>(`/documents/${docId}/review`, data);
        return response.data;
    }

    async deleteDocument(docId: number): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/documents/${docId}`);
        return response.data;
    }

    async getGeneratedDocuments(shipmentId: string): Promise<ApiResponse<GeneratedDocument[]>> {
        const response = await this.client.get<ApiResponse<GeneratedDocument[]>>(`/shipments/${shipmentId}/generated-documents`);
        return response.data;
    }

    async generatePackingList(shipmentId: string): Promise<ApiResponse<{ document: GeneratedDocument; download_url: string }>> {
        const response = await this.client.post<ApiResponse<{ document: GeneratedDocument; download_url: string }>>(`/shipments/${shipmentId}/generate/packing-list`);
        return response.data;
    }

    async generateInvoice(shipmentId: string): Promise<ApiResponse<{ document: GeneratedDocument; download_url: string }>> {
        const response = await this.client.post<ApiResponse<{ document: GeneratedDocument; download_url: string }>>(`/shipments/${shipmentId}/generate/invoice`);
        return response.data;
    }

    async getDocumentTemplates(): Promise<ApiResponse<DocumentTemplate[]>> {
        const response = await this.client.get<ApiResponse<DocumentTemplate[]>>('/documents/templates');
        return response.data;
    }

    async getOCRResults(docId: number): Promise<ApiResponse<OCRResult[]>> {
        const response = await this.client.get<ApiResponse<OCRResult[]>>(`/documents/${docId}/ocr-results`);
        return response.data;
    }

    async applyOCRResults(docId: number): Promise<ApiResponse<{ applied_count: number; updated_fields: Record<string, string> }>> {
        const response = await this.client.post<ApiResponse<{ applied_count: number; updated_fields: Record<string, string> }>>(`/documents/${docId}/apply-ocr`);
        return response.data;
    }

    // 预警
    async getAlerts(params?: { status?: string; severity?: string; type?: string }): Promise<ApiResponse<Alert[]>> {
        const response = await this.client.get<ApiResponse<Alert[]>>('/alerts', { params });
        return response.data;
    }

    async getAlert(id: string): Promise<ApiResponse<Alert>> {
        const response = await this.client.get<ApiResponse<Alert>>(`/alerts/${id}`);
        return response.data;
    }

    async resolveAlert(id: string): Promise<ApiResponse<Alert>> {
        const response = await this.client.post<ApiResponse<Alert>>(`/alerts/${id}/resolve`);
        return response.data;
    }

    // 用户
    async getUsers(params?: { status?: string; role?: string; search?: string }): Promise<ApiResponse<User[]>> {
        const response = await this.client.get<ApiResponse<User[]>>('/users', { params });
        return response.data;
    }

    async createUser(data: { email: string; password: string; name: string; role?: string }): Promise<ApiResponse<User>> {
        const response = await this.client.post<ApiResponse<User>>('/users', data);
        return response.data;
    }

    async updateUser(id: string, data: Partial<User>): Promise<ApiResponse<User>> {
        const response = await this.client.put<ApiResponse<User>>(`/users/${id}`, data);
        return response.data;
    }

    async deleteUser(id: string): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/users/${id}`);
        return response.data;
    }

    // 客户管理
    async getCustomers(params?: { type?: string; search?: string }): Promise<ApiResponse<Customer[]>> {
        const response = await this.client.get<ApiResponse<Customer[]>>('/customers', { params });
        return response.data;
    }

    async getCustomer(id: string): Promise<ApiResponse<Customer>> {
        const response = await this.client.get<ApiResponse<Customer>>(`/customers/${id}`);
        return response.data;
    }

    async createCustomer(data: Partial<Customer>): Promise<ApiResponse<Customer>> {
        const response = await this.client.post<ApiResponse<Customer>>('/customers', data);
        return response.data;
    }

    async updateCustomer(id: string, data: Partial<Customer>): Promise<ApiResponse<Customer>> {
        const response = await this.client.put<ApiResponse<Customer>>(`/customers/${id}`, data);
        return response.data;
    }

    async deleteCustomer(id: string): Promise<ApiResponse<void>> {
        const response = await this.client.delete<ApiResponse<void>>(`/customers/${id}`);
        return response.data;
    }

    async searchCustomers(params: { phone: string; type: string }): Promise<ApiResponse<Customer[]>> {
        const response = await this.client.get<ApiResponse<Customer[]>>('/customers/search', { params });
        return response.data;
    }

    async resetUserPassword(id: string, newPassword: string): Promise<ApiResponse<void>> {
        const response = await this.client.post<ApiResponse<void>>(`/users/${id}/reset-password`, { new_password: newPassword });
        return response.data;
    }

    // 仪表盘
    async getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
        const response = await this.client.get<ApiResponse<DashboardStats>>('/dashboard/stats');
        return response.data;
    }

    async getDashboardLocations(): Promise<ApiResponse<Device[]>> {
        const response = await this.client.get<ApiResponse<Device[]>>('/dashboard/locations');
        return response.data;
    }

    // 配置
    async getConfig(): Promise<ApiResponse<Record<string, string>>> {
        const response = await this.client.get<ApiResponse<Record<string, string>>>('/config');
        return response.data;
    }

    async updateConfig(data: Record<string, string>): Promise<ApiResponse<void>> {
        const response = await this.client.post<ApiResponse<void>>('/config', data);
        return response.data;
    }

    async testKuaihuoyunAPI(): Promise<ApiResponse<any>> {
        const response = await this.client.post<ApiResponse<any>>('/config/test-kuaihuoyun');
        return response.data;
    }

    // 运单字段配置
    async getShipmentFieldConfig(): Promise<ApiResponse<{
        bill_of_lading: boolean;
        container_no: boolean;
        seal_no: boolean;
        vessel_name: boolean;
        voyage_no: boolean;
        carrier: boolean;
        po_numbers: boolean;
        sku_ids: boolean;
        fba_shipment_id: boolean;
        surcharges: boolean;
        customs_fee: boolean;
        other_cost: boolean;
    }>> {
        const response = await this.client.get<ApiResponse<{
            bill_of_lading: boolean;
            container_no: boolean;
            seal_no: boolean;
            vessel_name: boolean;
            voyage_no: boolean;
            carrier: boolean;
            po_numbers: boolean;
            sku_ids: boolean;
            fba_shipment_id: boolean;
            surcharges: boolean;
            customs_fee: boolean;
            other_cost: boolean;
        }>>('/config/shipment-fields');
        return response.data;
    }

    async updateShipmentFieldConfig(data: {
        bill_of_lading?: boolean;
        container_no?: boolean;
        seal_no?: boolean;
        vessel_name?: boolean;
        voyage_no?: boolean;
        carrier?: boolean;
        po_numbers?: boolean;
        sku_ids?: boolean;
        fba_shipment_id?: boolean;
        surcharges?: boolean;
        customs_fee?: boolean;
        other_cost?: boolean;
    }): Promise<ApiResponse<void>> {
        const response = await this.client.put<ApiResponse<void>>('/config/shipment-fields', data);
        return response.data;
    }

    // 组织机构
    async getOrganizations(params?: { tree?: boolean; search?: string; parent_id?: string }): Promise<any> {
        const response = await this.client.get('/organizations', { params });
        return response.data;
    }

    async getOrganization(id: string): Promise<any> {
        const response = await this.client.get(`/organizations/${id}`);
        return response.data;
    }

    async createOrganization(data: {
        name: string;
        code: string;
        parent_id?: string;
        type?: string;
        sort?: number;
        leader_id?: string;
        description?: string;
    }): Promise<any> {
        const response = await this.client.post('/organizations', data);
        return response.data;
    }

    async updateOrganization(id: string, data: {
        name?: string;
        code?: string;
        type?: string;
        sort?: number;
        status?: string;
        leader_id?: string;
        description?: string;
    }): Promise<any> {
        const response = await this.client.put(`/organizations/${id}`, data);
        return response.data;
    }

    async deleteOrganization(id: string): Promise<any> {
        const response = await this.client.delete(`/organizations/${id}`);
        return response.data;
    }

    async moveOrganization(id: string, data: { parent_id?: string; sort?: number }): Promise<any> {
        const response = await this.client.put(`/organizations/${id}/move`, data);
        return response.data;
    }

    // 组织用户管理
    async getOrganizationUsers(id: string, params?: { include_sub?: boolean }): Promise<any> {
        const response = await this.client.get(`/organizations/${id}/users`, { params });
        return response.data;
    }

    async addUserToOrganization(orgId: string, data: {
        user_id: string;
        is_primary?: boolean;
        position?: string;
    }): Promise<any> {
        const response = await this.client.post(`/organizations/${orgId}/users`, data);
        return response.data;
    }

    async updateUserOrganization(orgId: string, userId: string, data: {
        is_primary?: boolean;
        position?: string;
    }): Promise<any> {
        const response = await this.client.put(`/organizations/${orgId}/users/${userId}`, data);
        return response.data;
    }

    async removeUserFromOrganization(orgId: string, userId: string): Promise<any> {
        const response = await this.client.delete(`/organizations/${orgId}/users/${userId}`);
        return response.data;
    }

    // 获取组织设备
    async getOrganizationDevices(id: string, params?: { include_sub?: boolean }): Promise<any> {
        const response = await this.client.get(`/organizations/${id}/devices`, { params });
        return response.data;
    }

    // 获取用户所属组织
    async getUserOrganizations(userId: string): Promise<any> {
        const response = await this.client.get(`/users/${userId}/organizations`);
        return response.data;
    }

    // 地理编码 (腾讯地图)
    async geocode(address: string, oversea?: boolean): Promise<{
        success: boolean;
        data: {
            address: string;
            short_name: string;
            lat: number;
            lng: number;
            province: string;
            city: string;
            district: string;
            nation: string;
            is_oversea: boolean;
            reliability: number;
        };
    }> {
        const response = await this.client.get('/geocode', {
            params: { address, oversea: oversea ? '1' : '0' }
        });
        return response.data;
    }

    // 地址输入联想 (腾讯地图)
    async addressSuggestion(keyword: string, region?: string, oversea?: boolean): Promise<{
        success: boolean;
        data: Array<{
            title: string;
            address: string;
            province: string;
            city: string;
            district: string;
            lat: number;
            lng: number;
        }>;
    }> {
        const response = await this.client.get('/geocode/suggestion', {
            params: { keyword, region, oversea: oversea ? '1' : '0' }
        });
        return response.data;
    }

    // 逆地理编码（坐标转地址）- 通过后端代理调用
    async reverseGeocode(lat: number, lng: number): Promise<{
        success: boolean;
        display_name: string;
    }> {
        const response = await this.client.get('/geocode/reverse', {
            params: { lat: lat.toFixed(6), lng: lng.toFixed(6) }
        });
        return response.data;
    }

    // 搜索建议
    async searchSuggestions(keyword: string): Promise<{
        suggestions: Array<{
            value: string;
            type: 'shipment' | 'device';
            id: string;
            label: string;
            subLabel: string;
        }>;
    }> {
        const response = await this.client.get('/search/suggestions', { params: { keyword } });
        return response.data;
    }

    // 通用 GET 请求方法
    async get<T = any>(url: string, config?: any): Promise<T> {
        const response = await this.client.get(url, config);
        return response.data;
    }

    // 通用 POST 请求方法
    async post<T = any>(url: string, data?: any, config?: any): Promise<T> {
        const response = await this.client.post(url, data, config);
        return response.data;
    }
}

export const api = new ApiClient();
export default api;
