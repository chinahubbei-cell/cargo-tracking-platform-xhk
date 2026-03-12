import request from '../utils/request'

export interface ShipmentFieldConfig {
    bill_of_lading: boolean
    container_no: boolean
    seal_no: boolean
    vessel_name: boolean
    voyage_no: boolean
    carrier: boolean
    po_numbers: boolean
    sku_ids: boolean
    fba_shipment_id: boolean
    surcharges: boolean
    customs_fee: boolean
    other_cost: boolean
}

export interface ShipmentTracksResponse {
    shipment_id: string
    device_ids: string[]
    tracks: Array<{
        id: number
        device_id: string
        lat: number
        lng: number
        speed: number
        direction: number
        temperature?: number
        humidity?: number
        locate_time: string
    }>
    current_position?: {
        lat: number
        lng: number
        speed?: number
        timestamp?: string
    } | null
}

export interface ShipmentStopRecord {
    id: string
    device_id: string
    device_external_id: string
    shipment_id: string
    start_time: string
    end_time?: string | null
    duration_seconds: number
    duration_text: string
    latitude?: number | null
    longitude?: number | null
    address: string
    status: 'active' | 'completed'
    alert_sent: boolean
    created_at: string
}

export interface ShipmentStopsResponse {
    records: ShipmentStopRecord[]
    total: number
    page: number
    page_size: number
}

export interface TransitCityNode {
    id: string
    country: string
    city: string
    latitude: number
    longitude: number
    entered_at: string
    is_oversea: boolean
}

export const AuthService = {
    login: (code: string) => request('/miniapp/auth/login', 'POST', { code }),
    bind: (data: any) => request('/miniapp/auth/bind', 'POST', data),
}

export const ShipmentService = {
    list: (params?: any) => request('/shipments', 'GET', params),
    get: (id: string) => request(`/shipments/${id}`, 'GET'),
    update: (id: string, data: any) => request(`/shipments/${id}`, 'PUT', data),
    getRoute: (id: string) => request(`/shipments/${id}/route`, 'GET'),
    getTracks: (id: string, params?: any) => request(`/shipments/${id}/tracks`, 'GET', params),
    getStops: (id: string, params?: { page?: number; page_size?: number }) => request(`/shipments/${id}/stops`, 'GET', params),
    getTransitCities: (id: string, refresh?: boolean) => request(`/shipments/${id}/transit-cities`, 'GET', refresh ? { refresh: 'true' } : undefined),
    create: (data: any) => request('/shipments', 'POST', data),
    bindDevice: (shipmentId: string, deviceId: string) => request(`/shipments/${shipmentId}`, 'PUT', { device_id: deviceId }),
    unbindDevice: (shipmentId: string) => request(`/shipments/${shipmentId}`, 'PUT', { device_id: '' }),
}

export const GeoService = {
    // 逆地理编码：坐标 → 结构化省市信息（与PC端 AddressInput 解析逻辑统一）
    reverseGeocode: (lat: number, lng: number) =>
        request<{ success: boolean; data: { display_name: string; province: string; city: string; district: string; country: string; short_name: string } }>(
            '/geocode/reverse', 'GET', { lat, lng }
        ),
}

export const ConfigService = {
    getShipmentFieldConfig: () => request<{ success?: boolean; data?: ShipmentFieldConfig } | ShipmentFieldConfig>('/config/shipment-fields', 'GET'),
}

export const DeviceService = {
    // Add device methods if needed
}
