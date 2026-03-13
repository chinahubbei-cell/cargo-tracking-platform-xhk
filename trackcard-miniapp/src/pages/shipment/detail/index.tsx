import React, { useEffect, useState } from 'react'
import { View, Text, Map } from '@tarojs/components'
import Taro, { useRouter } from '@tarojs/taro'
import { Button, Toast, Tag } from '@nutui/nutui-react-taro'
import {
    ConfigService,
    ShipmentFieldConfig,
    ShipmentService,
    ShipmentStopRecord,
    ShipmentTracksResponse,
    TransitCityNode,
} from '../../../services/api'
import './index.css'

interface Marker {
    id: number
    latitude: number
    longitude: number
    title: string
    iconPath: string
    width: number
    height: number
}

interface TransportNode {
    key: string
    type: 'stop' | 'city'
    timestamp: string
    locationKey: string
    description: string
}

interface ParsedLocation {
    country: string
    city: string
    label: string
    key: string
}

const COUNTRY_ALIASES: Array<{ pattern: RegExp; country: string }> = [
    { pattern: /中国|china|prc|people'?s republic of china/i, country: '中国' },
    { pattern: /美国|united states|usa|america/i, country: '美国' },
    { pattern: /俄罗斯|russia/i, country: '俄罗斯' },
    { pattern: /日本|japan/i, country: '日本' },
    { pattern: /德国|germany/i, country: '德国' },
    { pattern: /法国|france/i, country: '法国' },
]

const hasChinese = (text: string): boolean => /[\u4e00-\u9fa5]/.test(text)

const normalizeCountryName = (country: string): string => {
    const value = country.trim()
    if (!value) return ''
    for (const item of COUNTRY_ALIASES) {
        if (item.pattern.test(value)) {
            return item.country
        }
    }
    return value
}

const toTitleCase = (text: string): string => {
    return text
        .toLowerCase()
        .replace(/\b[a-z]/g, (ch) => ch.toUpperCase())
        .trim()
}

const normalizeCityName = (city: string): string => {
    let value = city.trim().replace(/^[,;.\s-]+|[,;.\s-]+$/g, '')
    if (!value) return ''

    if (hasChinese(value)) {
        value = value.replace(/\s+/g, '')
        if (!value || value === '市辖区') return ''
        if (value.startsWith('坐标区域(')) return value

        const cityMatches = value.match(/[\u4e00-\u9fa5]{2,16}(?:自治州|地区|盟|市)/g)
        if (cityMatches && cityMatches.length > 0) {
            return cityMatches[cityMatches.length - 1]
        }
        if (/(?:区|县|旗|省|特别行政区|自治区)$/.test(value)) {
            return value
        }
        return `${value}市`
    }

    value = value.replace(/\b(City|District|County|Prefecture|Province|State)\b$/i, '').trim()
    value = value.replace(/\s{2,}/g, ' ')
    return toTitleCase(value)
}

const extractCountryFromText = (text: string): string => {
    const value = text.trim()
    if (!value) return ''
    for (const item of COUNTRY_ALIASES) {
        if (item.pattern.test(value)) {
            return item.country
        }
    }
    return ''
}

const extractChineseCity = (text: string): string => {
    const value = text.trim()
    if (!value) return ''

    const cityMatch = value.match(/[\u4e00-\u9fa5]{2,12}(?:自治州|地区|盟|市)/)
    if (cityMatch?.[0]) {
        return cityMatch[0]
    }

    const districtMatch = value.match(/[\u4e00-\u9fa5]{2,12}(?:区|县|旗)/)
    return districtMatch?.[0] || ''
}

const extractEnglishCity = (text: string): string => {
    const value = text.trim()
    if (!value) return ''

    const segments = value
        .replace(/[，；;]/g, ',')
        .replace(/[()（）]/g, ' ')
        .split(',')
        .map((segment) => segment.trim())
        .filter(Boolean)

    for (const segment of segments) {
        const cityMatch = segment.match(/([A-Za-z][A-Za-z\s'-]{1,40})\s+City\b/i)
        if (cityMatch?.[1]) {
            return cityMatch[1].trim()
        }
    }

    for (const segment of segments) {
        const districtMatch = segment.match(/([A-Za-z][A-Za-z\s'-]{1,40})\s+(District|County|Prefecture|Province|State)\b/i)
        if (districtMatch?.[1]) {
            return districtMatch[1].trim()
        }
    }

    return ''
}

const resolveNodeLocation = (address: string, fallbackCountry = '', fallbackCity = ''): ParsedLocation => {
    const text = (address || '').trim()
    const parts = text.split('/')
    const zhPart = (parts[0] || '').trim()
    const enPart = (parts[1] || '').trim()

    let country = normalizeCountryName(fallbackCountry) || extractCountryFromText(`${zhPart} ${enPart}`)
    let city = extractChineseCity(zhPart)
    if (!city) city = extractEnglishCity(enPart)
    if (!city) city = extractEnglishCity(zhPart)
    if (!city) city = fallbackCity

    city = normalizeCityName(city)
    if (!country && city && hasChinese(city)) {
        country = '中国'
    }
    if (!country) {
        country = '未知国家'
    }
    if (!city) {
        city = '未知城市'
    }

    return {
        country,
        city,
        label: `${country}-${city}`,
        key: `${country.toLowerCase()}|${city.toLowerCase()}`,
    }
}

const defaultFieldConfig: ShipmentFieldConfig = {
    bill_of_lading: false,
    container_no: false,
    seal_no: false,
    vessel_name: false,
    voyage_no: false,
    carrier: false,
    po_numbers: false,
    sku_ids: false,
    fba_shipment_id: false,
    surcharges: false,
    customs_fee: false,
    other_cost: false,
}

const parseApiData = <T,>(res: any): T | undefined => {
    if (res && typeof res === 'object' && 'data' in res) {
        return res.data as T
    }
    return res as T
}

const parseShipment = (res: any) => {
    const payload = parseApiData<any>(res)
    if (payload && typeof payload === 'object' && payload.shipment) {
        return payload.shipment
    }
    return payload
}

const toNumber = (value: any): number | null => {
    const num = Number(value)
    return Number.isFinite(num) ? num : null
}

const isValidCoord = (lat: number | null, lng: number | null) => {
    return (
        lat !== null &&
        lng !== null &&
        lat >= -90 &&
        lat <= 90 &&
        lng >= -180 &&
        lng <= 180 &&
        !(lat === 0 && lng === 0)
    )
}

const formatDate = (dateStr: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date
        .getDate()
        .toString()
        .padStart(2, '0')}`
}

const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
    })
}

const parseDateSafe = (value: any): Date | null => {
    if (!value) return null
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return null
    return date
}

const formatDurationText = (seconds: number) => {
    if (seconds < 0) return '-'
    if (seconds < 60) return `${seconds}秒`
    if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60)
        return `${minutes}分钟`
    }
    if (seconds < 86400) {
        const hours = Math.floor(seconds / 3600)
        const minutes = Math.floor((seconds % 3600) / 60)
        return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
    }
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) {
        return minutes > 0 ? `${days}天${hours}小时${minutes}分钟` : `${days}天${hours}小时`
    }
    return minutes > 0 ? `${days}天${minutes}分钟` : `${days}天`
}

const getShipmentTotalDuration = (shipment: any) => {
    if (!shipment || typeof shipment !== 'object') return '-'

    const serverDuration = typeof shipment.total_duration === 'string' ? shipment.total_duration.trim() : ''
    if (serverDuration) return serverDuration

    const start =
        parseDateSafe(shipment.left_origin_at) ||
        parseDateSafe(shipment.atd) ||
        parseDateSafe(shipment.departure_time) ||
        parseDateSafe(shipment.device_bound_at) ||
        parseDateSafe(shipment.created_at)
    if (!start) return '-'

    const end = parseDateSafe(shipment.arrived_dest_at) || parseDateSafe(shipment.ata) || new Date()
    if (end.getTime() < start.getTime()) return '-'

    const totalSeconds = Math.floor((end.getTime() - start.getTime()) / 1000)
    return formatDurationText(totalSeconds)
}

function ShipmentDetail() {
    const router = useRouter()
    const { id } = router.params
    const [detail, setDetail] = useState<any>(null)
    const [markers, setMarkers] = useState<Marker[]>([])
    const [polyline, setPolyline] = useState<any[]>([])
    const [fieldConfig, setFieldConfig] = useState<ShipmentFieldConfig>(defaultFieldConfig)
    const [transportNodes, setTransportNodes] = useState<TransportNode[]>([])

    const fetchFieldConfig = async () => {
        try {
            const res: any = await ConfigService.getShipmentFieldConfig()
            const data = parseApiData<ShipmentFieldConfig>(res)
            if (data && typeof data === 'object') {
                setFieldConfig({
                    ...defaultFieldConfig,
                    ...data,
                })
            }
        } catch (err) {
            console.warn('加载运单字段配置失败，使用默认配置:', err)
        }
    }

    const updateMapData = (shipment: any, tracksPayload?: ShipmentTracksResponse | null) => {
        const originLat = toNumber(shipment?.origin_lat)
        const originLng = toNumber(shipment?.origin_lng)
        const destLat = toNumber(shipment?.dest_lat)
        const destLng = toNumber(shipment?.dest_lng)

        const currentLat = toNumber(tracksPayload?.current_position?.lat)
        const currentLng = toNumber(tracksPayload?.current_position?.lng)

        const mapMarkers: Marker[] = []
        if (isValidCoord(currentLat, currentLng)) {
            mapMarkers.push({
                id: 0,
                latitude: currentLat,
                longitude: currentLng,
                title: '当前位置',
                iconPath: '',
                width: 30,
                height: 30,
            })
        }
        if (isValidCoord(originLat, originLng)) {
            mapMarkers.push({
                id: 1,
                latitude: originLat,
                longitude: originLng,
                title: '起运地',
                iconPath: '',
                width: 20,
                height: 20,
            })
        }
        if (isValidCoord(destLat, destLng)) {
            mapMarkers.push({
                id: 2,
                latitude: destLat,
                longitude: destLng,
                title: '目的地',
                iconPath: '',
                width: 20,
                height: 20,
            })
        }
        setMarkers(mapMarkers)

        const routePoints: Array<{ latitude: number; longitude: number }> = []
        const tracks = Array.isArray(tracksPayload?.tracks) ? tracksPayload?.tracks : []
        tracks.forEach((track) => {
            const lat = toNumber(track.lat)
            const lng = toNumber(track.lng)
            if (isValidCoord(lat, lng)) {
                routePoints.push({ latitude: lat, longitude: lng })
            }
        })

        if (routePoints.length < 2 && isValidCoord(originLat, originLng) && isValidCoord(destLat, destLng)) {
            routePoints.push({ latitude: originLat, longitude: originLng })
            routePoints.push({ latitude: destLat, longitude: destLng })
        }

        if (routePoints.length >= 2) {
            setPolyline([
                {
                    points: routePoints,
                    color: '#1890ff',
                    width: 4,
                    dottedLine: false,
                },
            ])
        } else {
            setPolyline([])
        }
    }

    const fetchTracksAndMap = async (shipment: any) => {
        const shouldLoadTracks =
            !!shipment?.device_id ||
            !!shipment?.unbound_device_id ||
            shipment?.status === 'delivered' ||
            shipment?.status === 'cancelled'

        if (!shouldLoadTracks || !id) {
            updateMapData(shipment, null)
            return
        }

        try {
            const res: any = await ShipmentService.getTracks(id as string)
            const payload = parseApiData<ShipmentTracksResponse>(res)
            updateMapData(shipment, payload || null)
        } catch (err) {
            console.warn('加载轨迹失败，使用基础路线展示:', err)
            updateMapData(shipment, null)
        }
    }

    const fetchTransportNodes = async () => {
        if (!id) return

        const [stopsResult, transitResult] = await Promise.allSettled([
            ShipmentService.getStops(id as string, { page: 1, page_size: 100 }),
            ShipmentService.getTransitCities(id as string),
        ])

        let stops: ShipmentStopRecord[] = []
        if (stopsResult.status === 'fulfilled') {
            const stopPayload = parseApiData<any>(stopsResult.value)
            if (Array.isArray(stopPayload?.records)) {
                stops = stopPayload.records
            }
        }

        let transitCities: TransitCityNode[] = []
        if (transitResult.status === 'fulfilled') {
            const transitPayload = parseApiData<any>(transitResult.value)
            if (Array.isArray(transitPayload)) {
                transitCities = transitPayload
            }
        }

        // 仅“途径城市”按国家-城市去重；停留节点保留完整双语地址与原始记录粒度
        const stopNodes: TransportNode[] = stops
            .filter((record) => !!record.start_time)
            .map((record, index) => {
                const stopAddress = (record.address || '').trim() || '未知位置'
                const durationText = record.duration_text || '未知时长'
                return {
                    key: `stop-${record.id || index}`,
                    type: 'stop',
                    timestamp: record.start_time,
                    locationKey: `stop:${record.id || index}`,
                    description: `货物在：${stopAddress}停留，时长${durationText}`,
                }
            })

        const rawCityNodes: TransportNode[] = transitCities
            .filter((city) => !!city.entered_at)
            .map((city, index) => {
                const location = resolveNodeLocation('', city.country, city.city)
                return {
                    key: `city-${city.id || index}`,
                    type: 'city',
                    timestamp: city.entered_at,
                    locationKey: location.key,
                    description: `货物进入：${location.label}`,
                }
            })

        const cityNodes = rawCityNodes.sort((a, b) => {
            const aTime = new Date(a.timestamp).getTime()
            const bTime = new Date(b.timestamp).getTime()
            return (Number.isFinite(bTime) ? bTime : 0) - (Number.isFinite(aTime) ? aTime : 0)
        })

        const dedupedCityNodes: TransportNode[] = []
        const seenCityLocation = new Set<string>()
        for (const node of cityNodes) {
            if (seenCityLocation.has(node.locationKey)) {
                continue
            }
            seenCityLocation.add(node.locationKey)
            dedupedCityNodes.push(node)
        }

        const merged = [...stopNodes, ...dedupedCityNodes].sort((a, b) => {
            const aTime = new Date(a.timestamp).getTime()
            const bTime = new Date(b.timestamp).getTime()
            return (Number.isFinite(bTime) ? bTime : 0) - (Number.isFinite(aTime) ? aTime : 0)
        })

        setTransportNodes(merged)
    }

    const fetchDetail = async () => {
        if (!id) return

        try {
            const res: any = await ShipmentService.get(id as string)
            const shipment = parseShipment(res)

            if (!shipment || typeof shipment !== 'object') {
                throw new Error('运单数据格式异常')
            }

            setDetail(shipment)

            await Promise.all([fetchTracksAndMap(shipment), fetchTransportNodes()])
        } catch (err) {
            console.error(err)
            Taro.showToast({ title: '加载运单失败', icon: 'none' })
        }
    }

    useEffect(() => {
        if (!id) return

        fetchFieldConfig()
        fetchDetail()

        const timer = setInterval(() => {
            fetchDetail()
        }, 30000)

        return () => clearInterval(timer)
    }, [id])

    const showDocSection = fieldConfig.bill_of_lading || fieldConfig.container_no || fieldConfig.seal_no
    const showVesselVoyage = fieldConfig.vessel_name || fieldConfig.voyage_no

    // 从二维码内容中提取设备号（支持多种格式：URL、JSON、纯文本）
    const extractDeviceIdFromQR = (qrContent: string): string => {
        const content = (qrContent || '').trim()
        if (!content) return ''

        // 格式1: URL 带 imei/id/deviceId 参数 (如 https://xxx.com?imei=GC-f4855768)
        if (content.startsWith('http://') || content.startsWith('https://')) {
            try {
                const url = new URL(content)
                const params = url.searchParams
                const paramKeys = ['imei', 'id', 'deviceId', 'device_id', 'sn', 'devid']
                for (const key of paramKeys) {
                    const val = params.get(key)
                    if (val) return val
                }
                // URL 路径最后一段可能是设备号 (如 https://xxx.com/device/GC-f4855768)
                const pathParts = url.pathname.split('/').filter(Boolean)
                if (pathParts.length > 0) {
                    const lastPart = pathParts[pathParts.length - 1]
                    if (/^[A-Za-z0-9_-]{4,}$/.test(lastPart)) return lastPart
                }
            } catch (e) {
                console.warn('QR URL 解析失败:', e)
            }
        }

        // 格式2: JSON (如 {"deviceId": "GC-f4855768"})
        if (content.startsWith('{')) {
            try {
                const obj = JSON.parse(content)
                return obj.deviceId || obj.device_id || obj.imei || obj.id || obj.sn || ''
            } catch (e) {
                console.warn('QR JSON 解析失败:', e)
            }
        }

        // 格式3: 纯设备号文本 (如 GC-f4855768, 868120345678901)
        return content
    }

    const handleBindDevice = async () => {
        try {
            const res = await Taro.scanCode({ scanType: ['barCode', 'qrCode'] })
            console.log('[ScanCode] 扫码原始结果:', res.result)

            const deviceId = extractDeviceIdFromQR(res.result)
            console.log('[ScanCode] 提取设备号:', deviceId)

            if (!deviceId) {
                Taro.showToast({ title: '未识别到设备号', icon: 'none' })
                return
            }

            // 让用户确认设备号
            const confirmRes = await Taro.showModal({
                title: '确认绑定',
                content: `设备号: ${deviceId}\n确认绑定到当前运单？`,
                confirmText: '确认',
                cancelText: '取消',
            })
            if (!confirmRes.confirm) return

            Taro.showLoading({ title: '绑定中...' })
            await ShipmentService.bindDevice(id as string, deviceId)
            Taro.hideLoading()
            Taro.showToast({ title: '绑定成功', icon: 'success' })
            fetchDetail()
        } catch (err: any) {
            Taro.hideLoading()
            console.error('[ScanCode] 绑定失败:', err)
            if (err.errMsg && err.errMsg.indexOf('scanCode:fail') === -1) {
                Taro.showToast({ title: '绑定失败: ' + (err.message || ''), icon: 'none' })
            }
        }
    }

    const handleUnbindDevice = async () => {
        try {
            const modalRes = await Taro.showModal({
                title: '确认解绑',
                content: '确定要解绑当前设备吗？解绑后将无法追踪运单位置。',
                confirmText: '确定',
                cancelText: '取消',
            })

            if (!modalRes.confirm) {
                return
            }

            Taro.showLoading({ title: '解绑中...' })
            await ShipmentService.unbindDevice(id as string)
            Taro.hideLoading()
            Taro.showToast({ title: '解绑成功', icon: 'success' })
            fetchDetail()
        } catch (err: any) {
            Taro.hideLoading()
            console.error(err)
            Taro.showToast({ title: '解绑失败', icon: 'none' })
        }
    }

    const STATUS_MAP: Record<string, string> = {
        pending: '待发货',
        in_transit: '运输中',
        delivered: '已送达',
        cancelled: '已取消',
    }

    if (!detail) return <View>Loading...</View>

    const fallbackLat = toNumber(detail.origin_lat) ?? 39.9
    const fallbackLng = toNumber(detail.origin_lng) ?? 116.4
    const mapLatitude = markers.length > 0 ? markers[0].latitude : fallbackLat
    const mapLongitude = markers.length > 0 ? markers[0].longitude : fallbackLng

    // Helper to format historical device ID (removes GC- prefix if it's the only info we have)
    const formatUnboundDeviceId = (id: string) => {
        if (!id) return '';
        return id.startsWith('GC-') ? id.replace('GC-', '') : id;
    };

    const deviceDisplayId =
        detail.device?.external_device_id ||
        (detail.device?.id ? detail.device.id.replace(/^GC-/, '') : '') ||
        (detail.device_id ? detail.device_id.replace(/^GC-/, '') : '') ||
        (detail.unbound_device_id ? `${formatUnboundDeviceId(detail.unbound_device_id)} (已解绑)` : '未绑定')

    return (
        <View className="detail-container">
            <View className="detail-card">
                <View className="detail-header">
                    <Text className="detail-no">No. {detail.tracking_number || detail.id}</Text>
                    <Tag type={detail.status === 'delivered' ? 'success' : 'primary'}>
                        {STATUS_MAP[detail.status] || detail.status}
                    </Tag>
                </View>
                <View className="route-info">
                    <View className="route-node">
                        <Text className="node-city">{detail.origin || detail.origin_address || '-'}</Text>
                        <Text className="node-label">发货地</Text>
                    </View>
                    <View className="route-arrow">--&gt;</View>
                    <View className="route-node">
                        <Text className="node-city">{detail.destination || detail.dest_address || '-'}</Text>
                        <Text className="node-label">目的地</Text>
                    </View>
                </View>
            </View>

            <View className="info-section">
                <Text className="info-title">货物信息</Text>
                <View className="info-grid">
                    <View className="info-item full-width">
                        <View className="info-label">货物名称</View>
                        <View className="info-value">{detail.cargo_name || '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">件数</View>
                        <View className="info-value">{detail.pieces || '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">重量 (kg)</View>
                        <View className="info-value">{detail.weight || '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">体积 (m³)</View>
                        <View className="info-value">{detail.volume || '-'}</View>
                    </View>
                </View>
            </View>

            {showDocSection && (
                <View className="info-section">
                    <Text className="info-title">关键单证</Text>
                    <View className="info-grid">
                        {fieldConfig.bill_of_lading && (
                            <View className="info-item full-width">
                                <View className="info-label">提单号</View>
                                <View className="info-value">{detail.bill_of_lading || '-'}</View>
                            </View>
                        )}
                        {fieldConfig.container_no && (
                            <View className="info-item">
                                <View className="info-label">箱号</View>
                                <View className="info-value">{detail.container_no || '-'}</View>
                            </View>
                        )}
                        {fieldConfig.seal_no && (
                            <View className="info-item">
                                <View className="info-label">封条号</View>
                                <View className="info-value">{detail.seal_no || '-'}</View>
                            </View>
                        )}
                    </View>
                </View>
            )}

            <View className="info-section">
                <Text className="info-title">船务信息</Text>
                <View className="info-grid">
                    <View className="info-item">
                        <View className="info-label">运输方式</View>
                        <View className="info-value">
                            {{ sea: '海运', air: '空运', land: '陆运', multimodal: '多式联运' }[detail.transport_type] ||
                                detail.transport_type ||
                                '-'}
                        </View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">运输模式</View>
                        <View className="info-value">
                            {{ lcl: '零担', fcl: '整柜' }[detail.transport_mode] || detail.transport_mode || '-'}
                        </View>
                        {detail.transport_mode === 'fcl' && detail.container_type && (
                            <Text style={{ marginLeft: '4px', color: '#666' }}>({detail.container_type})</Text>
                        )}
                    </View>
                    {showVesselVoyage && (
                        <View className="info-item">
                            <View className="info-label">船名/航次</View>
                            <View className="info-value">
                                {fieldConfig.vessel_name ? detail.vessel_name || '-' : '-'} /
                                {fieldConfig.voyage_no ? detail.voyage_no || '-' : '-'}
                            </View>
                        </View>
                    )}
                    {fieldConfig.carrier && (
                        <View className="info-item">
                            <View className="info-label">承运人</View>
                            <View className="info-value">{detail.carrier || '-'}</View>
                        </View>
                    )}
                    <View className="info-item full-width">
                        <View className="info-label">定位设备ID</View>
                        <View className="info-value">{deviceDisplayId}</View>
                    </View>
                </View>
            </View>

            <View className="info-section">
                <Text className="info-title">时间节点</Text>
                <View className="info-grid">
                    <View className="info-item">
                        <View className="info-label">预计出发 (ETD)</View>
                        <View className="info-value">{detail.etd ? formatDate(detail.etd) : '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">实际出发 (ATD)</View>
                        <View className="info-value">{detail.atd ? formatDate(detail.atd) : '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">预计到达 (ETA)</View>
                        <View className="info-value">{detail.eta ? formatDate(detail.eta) : '-'}</View>
                    </View>
                    <View className="info-item">
                        <View className="info-label">实际到达 (ATA)</View>
                        <View className="info-value">{detail.ata ? formatDate(detail.ata) : '-'}</View>
                    </View>
                    <View className="info-item full-width total-duration-item">
                        <View className="info-label">当前运单总耗时</View>
                        <View className="info-value total-duration-value">{getShipmentTotalDuration(detail)}</View>
                    </View>
                </View>
            </View>

            <View className="map-section">
                <Map
                    id="map"
                    style={{ width: '100%', height: '300px' }}
                    latitude={mapLatitude}
                    longitude={mapLongitude}
                    scale={5}
                    markers={markers}
                    polyline={polyline}
                    showLocation
                />
            </View>

            <View className="timeline-section">
                <Text className="section-title">运输节点信息</Text>
                {transportNodes.length === 0 ? (
                    <View className="timeline-empty">暂无运输节点信息</View>
                ) : (
                    <View className="timeline-list">
                        {transportNodes.map((node) => (
                            <View key={node.key} className="timeline-node-item">
                                <View className={`timeline-node-dot ${node.type === 'city' ? 'city' : 'stop'}`}>
                                    <Text className="timeline-node-dot-text">{node.type === 'city' ? '城' : '停'}</Text>
                                </View>
                                <View className="timeline-node-main">
                                    <Text className="timeline-node-desc">{node.description}</Text>
                                    <Text className="timeline-node-time">{formatDateTime(node.timestamp)}</Text>
                                </View>
                            </View>
                        ))}
                    </View>
                )}
            </View>

            <View className="floating-device-bar">
                {detail.device_id ? (
                    <View className="floating-device-content bound">
                        <View className="floating-device-text">
                            <Text className="floating-device-title">已绑定设备ID</Text>
                            <Text className="floating-device-id">{deviceDisplayId}</Text>
                        </View>
                        {['delivered', 'cancelled'].includes(detail.status) ? null : (
                            <Button type="warning" size="small" onClick={handleUnbindDevice}>
                                解绑
                            </Button>
                        )}
                    </View>
                ) : ['delivered', 'cancelled'].includes(detail.status) ? (
                    <View className="floating-device-content bound">
                        <View className="floating-device-text">
                            <Text className="floating-device-title">{detail.unbound_device_id ? '历史绑定设备' : '未绑定设备'}</Text>
                            <Text className="floating-device-id">{deviceDisplayId}</Text>
                        </View>
                    </View>
                ) : (
                    <View className="floating-device-content unbound">
                        <View className="floating-device-text">
                            <Text className="floating-device-title">未绑定设备ID</Text>
                            <Text className="floating-device-subtitle">支持扫码自动绑定设备号并同步轨迹</Text>
                        </View>
                        <Button type="primary" size="small" className="floating-bind-button" onClick={handleBindDevice}>
                            扫码绑定
                        </Button>
                    </View>
                )}
            </View>

            <Toast />
        </View>
    )
}

export default ShipmentDetail
