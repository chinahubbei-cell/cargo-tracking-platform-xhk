import React, { useEffect, useState } from 'react'
import { View } from '@tarojs/components'
import Taro, { useRouter } from '@tarojs/taro'
import { Button, Input, Picker, DatePicker } from '@nutui/nutui-react-taro'
import { ConfigService, GeoService, ShipmentFieldConfig, ShipmentService } from '../../../services/api'
import './index.css'

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

// 纯布局的表单行组件（不使用 NutUI Form，避免值串联）
function FormRow({ label, required, onClick, children }: {
    label: string; required?: boolean; onClick?: () => void; children: React.ReactNode
}) {
    return (
        <View className="form-row" onClick={onClick}>
            <View className="form-row-label">
                {required && <View style={{ color: '#ff4d4f', display: 'inline' }}>* </View>}
                {label}
            </View>
            <View className="form-row-value">
                {children}
            </View>
        </View>
    )
}

function CreateShipment() {
    const [loading, setLoading] = useState(false)
    const [fieldConfig, setFieldConfig] = useState<ShipmentFieldConfig>(defaultFieldConfig)

    // ========== 用 useState 直接管理所有表单字段 ==========
    const [senderName, setSenderName] = useState('')
    const [senderPhone, setSenderPhone] = useState('')
    const [receiverName, setReceiverName] = useState('')
    const [receiverPhone, setReceiverPhone] = useState('')
    const [deviceId, setDeviceId] = useState('')
    const [cargoName, setCargoName] = useState('')
    const [pieces, setPieces] = useState('')
    const [weight, setWeight] = useState('')
    const [volume, setVolume] = useState('')
    const [billOfLading, setBillOfLading] = useState('')
    const [containerNo, setContainerNo] = useState('')
    const [sealNo, setSealNo] = useState('')
    const [vesselName, setVesselName] = useState('')
    const [voyageNo, setVoyageNo] = useState('')
    const [carrierVal, setCarrierVal] = useState('')
    const [poNumbers, setPoNumbers] = useState('')
    const [skuIds, setSkuIds] = useState('')
    const [fbaShipmentId, setFbaShipmentId] = useState('')
    const [freightCost, setFreightCost] = useState('')
    const [surchargesVal, setSurchargesVal] = useState('')
    const [customsFee, setCustomsFee] = useState('')
    const [otherCost, setOtherCost] = useState('')

    // Pickers State
    const [showTransportType, setShowTransportType] = useState(false)
    const [transportTypeDesc, setTransportTypeDesc] = useState('')
    const [transportTypeVal, setTransportTypeVal] = useState('')
    const transportTypeOptions = [
        { value: 'sea', text: '🚢 海运' },
        { value: 'air', text: '✈️ 空运' },
        { value: 'land', text: '🚚 陆运' },
        { value: 'multimodal', text: '🔄 多式联运' },
    ]

    const [showTransportMode, setShowTransportMode] = useState(false)
    const [transportModeDesc, setTransportModeDesc] = useState('')
    const [transportModeVal, setTransportModeVal] = useState('')
    const transportModeOptions = [
        { value: 'lcl', text: '📦 零担' },
        { value: 'fcl', text: '🚢 整柜' },
    ]

    const [showContainerType, setShowContainerType] = useState(false)
    const [containerTypeDesc, setContainerTypeDesc] = useState('')
    const [containerTypeVal, setContainerTypeVal] = useState('')
    const containerTypeOptions = [
        { value: '20GP', text: '20GP' },
        { value: '40GP', text: '40GP' },
        { value: '40HQ', text: '40HQ' },
        { value: '45HQ', text: '45HQ' },
    ]

    const [showCargoType, setShowCargoType] = useState(false)
    const [cargoTypeDesc, setCargoTypeDesc] = useState('')
    const [cargoTypeVal, setCargoTypeVal] = useState('')
    const cargoTypeOptions = [
        { value: 'general', text: '普货' },
        { value: 'dangerous', text: '危险品' },
        { value: 'cold_chain', text: '冷链' },
    ]

    // Date Pickers State
    const [showEtd, setShowEtd] = useState(false)
    const [etdDesc, setEtdDesc] = useState('')
    const [showEta, setShowEta] = useState(false)
    const [etaDesc, setEtaDesc] = useState('')

    // Location State
    const [originLoc, setOriginLoc] = useState<{ lat: number, lng: number } | null>(null)
    const [destLoc, setDestLoc] = useState<{ lat: number, lng: number } | null>(null)
    const [originShortName, setOriginShortName] = useState('')
    const [destShortName, setDestShortName] = useState('')
    const [originAddress, setOriginAddress] = useState('')
    const [destAddress, setDestAddress] = useState('')

    const hasOrderFields = fieldConfig.po_numbers || fieldConfig.sku_ids || fieldConfig.fba_shipment_id
    const router = useRouter()

    useEffect(() => {
        const loadFieldConfig = async () => {
            try {
                const res: any = await ConfigService.getShipmentFieldConfig()
                const data = res?.data ?? res
                if (data && typeof data === 'object') {
                    setFieldConfig({ ...defaultFieldConfig, ...data })
                }
            } catch (err) {
                console.warn('加载运单字段配置失败，使用默认配置:', err)
            }
        }
        loadFieldConfig()

        const deviceIdParam = router.params?.deviceId
        if (deviceIdParam) {
            const decodedId = decodeURIComponent(deviceIdParam)
            setDeviceId(decodedId)
            Taro.showToast({ title: `已填充设备: ${decodedId}`, icon: 'none', duration: 2000 })
        }
    }, [])

    const confirmPicker = (value: any[], options: any[], descSetter: any, valSetter: any) => {
        const option = options.find(o => o.value === value[0])
        if (option) {
            descSetter(option.text)
            valSetter(option.value)
        }
    }

    const confirmDate = (values: any[], descSetter: any) => {
        descSetter(values.join('-'))
    }

    const extractLocationShortName = (address: string, name: string) => {
        const match = address.match(/^([^省]+?[省份|自治区|特别行政区])?([^市]+?[州市])/)
        if (match) return (match[1] || '') + (match[2] || '')
        if (name) return name
        if (address && address.includes(' ')) return address.split(' ')[0]
        return address.slice(0, 4)
    }

    const handleChooseLocation = async (type: 'origin' | 'dest') => {
        try {
            let currentOptions: Taro.chooseLocation.Option = {}
            try {
                const locRes = await Taro.getLocation({ type: 'gcj02' })
                if (locRes?.latitude && locRes?.longitude) {
                    currentOptions = { latitude: locRes.latitude, longitude: locRes.longitude }
                }
            } catch (err) {
                console.log('未能获取当前位置:', err)
            }

            const res = await Taro.chooseLocation(currentOptions)
            if (res?.latitude && res?.longitude) {
                const locationData = { lat: res.latitude, lng: res.longitude }
                const fullAddress = res.address + (res.name ? ` (${res.name})` : '')
                const fallbackShortName = extractLocationShortName(res.address || '', res.name || '')

                if (type === 'origin') {
                    setOriginLoc(locationData)
                    setOriginShortName(fallbackShortName)
                    setOriginAddress(fullAddress)
                } else {
                    setDestLoc(locationData)
                    setDestShortName(fallbackShortName)
                    setDestAddress(fullAddress)
                }

                GeoService.reverseGeocode(res.latitude, res.longitude)
                    .then((geoRes: any) => {
                        const shortName = (geoRes?.data || geoRes)?.short_name || ''
                        if (shortName) {
                            if (type === 'origin') setOriginShortName(shortName)
                            else setDestShortName(shortName)
                        }
                    })
                    .catch(() => { /* 已用正则兜底 */ })
            }
        } catch (error) {
            console.log('取消选择位置或出错', error)
        }
    }

    const submit = async () => {
        // 必填校验
        if (!senderName.trim()) { Taro.showToast({ title: '请填写发货人', icon: 'none' }); return }
        if (!senderPhone.trim()) { Taro.showToast({ title: '请填写发货电话', icon: 'none' }); return }
        if (!originAddress) { Taro.showToast({ title: '请选择发货地址', icon: 'none' }); return }
        if (!receiverName.trim()) { Taro.showToast({ title: '请填写收货人', icon: 'none' }); return }
        if (!receiverPhone.trim()) { Taro.showToast({ title: '请填写收货电话', icon: 'none' }); return }
        if (!destAddress) { Taro.showToast({ title: '请选择收货地址', icon: 'none' }); return }
        if (!transportTypeVal) { Taro.showToast({ title: '请选择运输类型', icon: 'none' }); return }

        try {
            setLoading(true)
            const payload: any = {
                etd: etdDesc ? new Date(etdDesc).toISOString() : undefined,
                eta: etaDesc ? new Date(etaDesc).toISOString() : undefined,
                origin: originShortName || extractLocationShortName(originAddress, ''),
                destination: destShortName || extractLocationShortName(destAddress, ''),
                origin_lat: originLoc?.lat ? Number(originLoc.lat) : undefined,
                origin_lng: originLoc?.lng ? Number(originLoc.lng) : undefined,
                origin_address: originAddress,
                dest_lat: destLoc?.lat ? Number(destLoc.lat) : undefined,
                dest_lng: destLoc?.lng ? Number(destLoc.lng) : undefined,
                dest_address: destAddress,
                auto_status_enabled: true,
                origin_radius: 1000,
                dest_radius: 1000,
                origin_port_code: undefined,
                dest_port_code: undefined,
                sender_name: senderName.trim(),
                sender_phone: senderPhone.trim(),
                receiver_name: receiverName.trim(),
                receiver_phone: receiverPhone.trim(),
                transport_type: transportTypeVal || undefined,
                transport_mode: transportModeVal || 'lcl',
                container_type: containerTypeVal || undefined,
                cargo_type: cargoTypeVal || 'general',
                device_id: deviceId.trim() || undefined,
                cargo_name: cargoName.trim() || '',
                pieces: pieces ? Number(pieces) : undefined,
                weight: weight ? Number(weight) : undefined,
                volume: volume ? Number(volume) : undefined,
                bill_of_lading: billOfLading.trim() || undefined,
                container_no: containerNo.trim() || undefined,
                seal_no: sealNo.trim() || undefined,
                vessel_name: vesselName.trim() || undefined,
                voyage_no: voyageNo.trim() || undefined,
                carrier: carrierVal.trim() || undefined,
                po_numbers: poNumbers.trim() || undefined,
                sku_ids: skuIds.trim() || undefined,
                fba_shipment_id: fbaShipmentId.trim() || undefined,
                freight_cost: freightCost ? Number(freightCost) : undefined,
                surcharges: surchargesVal ? Number(surchargesVal) : undefined,
                customs_fee: customsFee ? Number(customsFee) : undefined,
                other_cost: otherCost ? Number(otherCost) : undefined,
                total_cost: (freightCost ? Number(freightCost) : 0) +
                            (surchargesVal ? Number(surchargesVal) : 0) +
                            (customsFee ? Number(customsFee) : 0) +
                            (otherCost ? Number(otherCost) : 0),
                status: 'pending'
            }

            // 清理 undefined 和 NaN
            Object.keys(payload).forEach(key => {
                if (payload[key] === undefined || Number.isNaN(payload[key])) {
                    delete payload[key]
                }
            })

            console.log('===== Final Payload =====', JSON.stringify(payload))
            await ShipmentService.create(payload)
            Taro.showToast({ title: '🎉 运单创建成功！', icon: 'success', duration: 2000 })
            setTimeout(() => Taro.navigateBack(), 2000)
        } catch (err: any) {
            console.error('Submit error:', err)
            Taro.showToast({ title: '创建失败: ' + (err.message || '未知错误'), icon: 'none' })
        } finally {
            setLoading(false)
        }
    }

    return (
        <View className="create-container">
            {/* 发货地与目的地展示 */}
            <View className="route-summary">
                <View className="route-item">
                    <View className="route-value">{originShortName || '待获取'}</View>
                    <View className="route-label">发货地</View>
                </View>
                <View className="route-arrow">→</View>
                <View className="route-item">
                    <View className="route-value">{destShortName || '待获取'}</View>
                    <View className="route-label">目的地</View>
                </View>
            </View>

            {/* 1. 路线与时效 */}
            <View className="section-title">📍 路线与时效</View>
            <FormRow label="预计出发" onClick={() => setShowEtd(true)}>
                <View className="form-row-display">{etdDesc || '请选择ETD'}</View>
            </FormRow>
            <FormRow label="预计到达" onClick={() => setShowEta(true)}>
                <View className="form-row-display">{etaDesc || '请选择ETA'}</View>
            </FormRow>

            {/* 发货方信息 */}
            <FormRow label="发货人" required>
                <Input placeholder="姓名" align="right" value={senderName}
                    onChange={(v) => setSenderName(String(v))} />
            </FormRow>
            <FormRow label="发货电话" required>
                <Input placeholder="电话" align="right" value={senderPhone}
                    onChange={(v) => setSenderPhone(String(v))} />
            </FormRow>
            <FormRow label="发货地址" required onClick={() => handleChooseLocation('origin')}>
                <View className="form-row-display">{originAddress || '点击选择发货地址...'}</View>
            </FormRow>

            <View style={{ height: '12px', backgroundColor: '#f5f6f7', margin: '10px -16px' }} />

            {/* 收货方信息 */}
            <FormRow label="收货人" required>
                <Input placeholder="姓名" align="right" value={receiverName}
                    onChange={(v) => setReceiverName(String(v))} />
            </FormRow>
            <FormRow label="收货电话" required>
                <Input placeholder="电话" align="right" value={receiverPhone}
                    onChange={(v) => setReceiverPhone(String(v))} />
            </FormRow>
            <FormRow label="收货地址" required onClick={() => handleChooseLocation('dest')}>
                <View className="form-row-display">{destAddress || '点击选择收货地址...'}</View>
            </FormRow>

            {/* 2. 运输配置 */}
            <View className="section-title">🚚 运输配置</View>
            <FormRow label="运输类型" required onClick={() => setShowTransportType(true)}>
                <View className="form-row-display">{transportTypeDesc || '请选择'}</View>
            </FormRow>
            <FormRow label="运输模式" onClick={() => setShowTransportMode(true)}>
                <View className="form-row-display">{transportModeDesc || '请选择'}</View>
            </FormRow>
            {transportModeVal === 'fcl' && (
                <FormRow label="柜型" onClick={() => setShowContainerType(true)}>
                    <View className="form-row-display">{containerTypeDesc || '请选择'}</View>
                </FormRow>
            )}
            <FormRow label="货物类型" onClick={() => setShowCargoType(true)}>
                <View className="form-row-display">{cargoTypeDesc || '请选择'}</View>
            </FormRow>
            <FormRow label="设备ID">
                <Input placeholder="输入设备ID" align="right" value={deviceId}
                    onChange={(v) => setDeviceId(String(v))} />
            </FormRow>

            {/* 3. 货物与单证 */}
            <View className="section-title">📦 货物与单证</View>
            <FormRow label="货物名称">
                <Input placeholder="输入货物名称" align="right" value={cargoName}
                    onChange={(v) => setCargoName(String(v))} />
            </FormRow>
            <FormRow label="件数">
                <Input placeholder="0" type="number" align="right" value={pieces}
                    onChange={(v) => setPieces(String(v))} />
            </FormRow>
            <FormRow label="重量(kg)">
                <Input placeholder="0.0" type="number" align="right" value={weight}
                    onChange={(v) => setWeight(String(v))} />
            </FormRow>
            <FormRow label="体积(m³)">
                <Input placeholder="0.0" type="number" align="right" value={volume}
                    onChange={(v) => setVolume(String(v))} />
            </FormRow>
            {fieldConfig.bill_of_lading && (
                <FormRow label="提单号">
                    <Input placeholder="MBL/HBL/AWB" align="right" value={billOfLading}
                        onChange={(v) => setBillOfLading(String(v))} />
                </FormRow>
            )}
            {fieldConfig.container_no && (
                <FormRow label="箱号">
                    <Input placeholder="输入箱号" align="right" value={containerNo}
                        onChange={(v) => setContainerNo(String(v))} />
                </FormRow>
            )}
            {fieldConfig.seal_no && (
                <FormRow label="封条号">
                    <Input placeholder="输入封条号" align="right" value={sealNo}
                        onChange={(v) => setSealNo(String(v))} />
                </FormRow>
            )}
            {fieldConfig.vessel_name && (
                <FormRow label="船名">
                    <Input placeholder="输入船名" align="right" value={vesselName}
                        onChange={(v) => setVesselName(String(v))} />
                </FormRow>
            )}
            {fieldConfig.voyage_no && (
                <FormRow label="航次">
                    <Input placeholder="输入航次" align="right" value={voyageNo}
                        onChange={(v) => setVoyageNo(String(v))} />
                </FormRow>
            )}
            {fieldConfig.carrier && (
                <FormRow label="船司/航司">
                    <Input placeholder="输入船司/航司" align="right" value={carrierVal}
                        onChange={(v) => setCarrierVal(String(v))} />
                </FormRow>
            )}

            {hasOrderFields && <View className="section-title">📋 订单关联</View>}
            {fieldConfig.po_numbers && (
                <FormRow label="PO单号">
                    <Input placeholder="输入PO单号" align="right" value={poNumbers}
                        onChange={(v) => setPoNumbers(String(v))} />
                </FormRow>
            )}
            {fieldConfig.sku_ids && (
                <FormRow label="SKU ID">
                    <Input placeholder="输入SKU ID" align="right" value={skuIds}
                        onChange={(v) => setSkuIds(String(v))} />
                </FormRow>
            )}
            {fieldConfig.fba_shipment_id && (
                <FormRow label="FBA编号">
                    <Input placeholder="输入FBA编号" align="right" value={fbaShipmentId}
                        onChange={(v) => setFbaShipmentId(String(v))} />
                </FormRow>
            )}

            {/* 4. 费用信息 */}
            <View className="section-title">💰 费用信息</View>
            <FormRow label="运费">
                <Input placeholder="0.00" type="digit" align="right" value={freightCost}
                    onChange={(v) => setFreightCost(String(v))} />
            </FormRow>
            {fieldConfig.surcharges && (
                <FormRow label="附加费">
                    <Input placeholder="0.00" type="digit" align="right" value={surchargesVal}
                        onChange={(v) => setSurchargesVal(String(v))} />
                </FormRow>
            )}
            {fieldConfig.customs_fee && (
                <FormRow label="关税">
                    <Input placeholder="0.00" type="digit" align="right" value={customsFee}
                        onChange={(v) => setCustomsFee(String(v))} />
                </FormRow>
            )}
            {fieldConfig.other_cost && (
                <FormRow label="其他费用">
                    <Input placeholder="0.00" type="digit" align="right" value={otherCost}
                        onChange={(v) => setOtherCost(String(v))} />
                </FormRow>
            )}

            {/* Pickers */}
            <Picker visible={showTransportType} options={transportTypeOptions}
                onConfirm={(_, values) => { confirmPicker(values, transportTypeOptions, setTransportTypeDesc, setTransportTypeVal); setShowTransportType(false) }}
                onClose={() => setShowTransportType(false)} />
            <Picker visible={showTransportMode} options={transportModeOptions}
                onConfirm={(_, values) => { confirmPicker(values, transportModeOptions, setTransportModeDesc, setTransportModeVal); setShowTransportMode(false) }}
                onClose={() => setShowTransportMode(false)} />
            <Picker visible={showContainerType} options={containerTypeOptions}
                onConfirm={(_, values) => { confirmPicker(values, containerTypeOptions, setContainerTypeDesc, setContainerTypeVal); setShowContainerType(false) }}
                onClose={() => setShowContainerType(false)} />
            <Picker visible={showCargoType} options={cargoTypeOptions}
                onConfirm={(_, values) => { confirmPicker(values, cargoTypeOptions, setCargoTypeDesc, setCargoTypeVal); setShowCargoType(false) }}
                onClose={() => setShowCargoType(false)} />
            <DatePicker visible={showEtd} type="date"
                defaultValue={etdDesc ? new Date(etdDesc) : new Date()}
                onConfirm={(_, values) => { if (values) confirmDate(values, setEtdDesc); setShowEtd(false) }}
                onClose={() => setShowEtd(false)} />
            <DatePicker visible={showEta} type="date"
                defaultValue={etaDesc ? new Date(etaDesc) : new Date()}
                onConfirm={(_, values) => { if (values) confirmDate(values, setEtaDesc); setShowEta(false) }}
                onClose={() => setShowEta(false)} />

            <Button type="primary" block loading={loading} onClick={submit}
                style={{ marginTop: '20px', marginBottom: '40px' }}>
                提交运单
            </Button>
        </View>
    )
}

export default CreateShipment
