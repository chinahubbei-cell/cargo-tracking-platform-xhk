import React, { useEffect, useState } from 'react'
import { View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { Button, Input, Form, Toast, Picker, DatePicker } from '@nutui/nutui-react-taro'
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

function CreateShipment() {
    const [loading, setLoading] = useState(false)
    const [form] = Form.useForm()
    const [fieldConfig, setFieldConfig] = useState<ShipmentFieldConfig>(defaultFieldConfig)

    // Pickers State
    const [showTransportType, setShowTransportType] = useState(false)
    const [transportTypeDesc, setTransportTypeDesc] = useState('')
    const transportTypeOptions = [
        { value: 'sea', text: '🚢 海运' },
        { value: 'air', text: '✈️ 空运' },
        { value: 'land', text: '🚚 陆运' },
        { value: 'multimodal', text: '🔄 多式联运' },
    ]

    const [showTransportMode, setShowTransportMode] = useState(false)
    const [transportModeDesc, setTransportModeDesc] = useState('')
    const transportModeOptions = [
        { value: 'lcl', text: '📦 零担' },
        { value: 'fcl', text: '🚢 整柜' },
    ]

    const [showContainerType, setShowContainerType] = useState(false)
    const [containerTypeDesc, setContainerTypeDesc] = useState('')
    const containerTypeOptions = [
        { value: '20GP', text: '20GP' },
        { value: '40GP', text: '40GP' },
        { value: '40HQ', text: '40HQ' },
        { value: '45HQ', text: '45HQ' },
    ]

    const [showCargoType, setShowCargoType] = useState(false)
    const [cargoTypeDesc, setCargoTypeDesc] = useState('')
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

    const hasOrderFields = fieldConfig.po_numbers || fieldConfig.sku_ids || fieldConfig.fba_shipment_id

    useEffect(() => {
        const loadFieldConfig = async () => {
            try {
                const res: any = await ConfigService.getShipmentFieldConfig()
                const data = res?.data ?? res
                if (data && typeof data === 'object') {
                    setFieldConfig({
                        ...defaultFieldConfig,
                        ...data
                    })
                }
            } catch (err) {
                console.warn('加载运单字段配置失败，使用默认配置:', err)
            }
        }

        loadFieldConfig()
    }, [])

    const confirmPicker = (fieldName: string, options: any[], value: any[], descSetter: any) => {
        const option = options.find(o => o.value === value[0])
        const text = option ? option.text : ''
        descSetter(text)
        form.setFieldsValue({ [fieldName]: text }) // Keep the text value in form to display correctly in Form.Item
    }

    const parseOptionValue = (options: any[], text: string) => {
        const option = options.find(o => o.text === text)
        return option ? option.value : text
    }

    const confirmDate = (values: any[], descSetter: any, fieldName: string) => {
        const dateStr = values.join('-')
        descSetter(dateStr)
        form.setFieldsValue({ [fieldName]: dateStr })
    }

    const extractLocationShortName = (address: string, name: string) => {
        // 先检查是否包含省市等关键字
        const provinceCityRegex = /^([^省]+?[省份|自治区|特别行政区])?([^市]+?[州市])/;
        const match = address.match(provinceCityRegex);
        
        if (match) {
            // 如果能用正则提取出省和市(含直辖市的特殊情况)，拼接省+市
            const province = match[1] || '';
            const city = match[2] || '';
            return province + city;
        }
        
        // 兜底一：没有市级关键字，截取名字或者按空格切分发货地
        if (name) return name;
        if (address && address.includes(' ')) return address.split(' ')[0];
        // 兜底二
        return address.slice(0, 4); // 最后防卫
    }

    const handleChooseLocation = async (type: 'origin' | 'dest') => {
        try {
            // 首先尝试获取当前位置作为地图默认中心点
            let currentOptions: Taro.chooseLocation.Option = {}
            try {
                const locRes = await Taro.getLocation({ type: 'gcj02' })
                if (locRes && locRes.latitude && locRes.longitude) {
                    currentOptions = { latitude: locRes.latitude, longitude: locRes.longitude }
                }
            } catch (err) {
                console.log('未能获取当前位置，使用默认行为:', err)
            }

            const res = await Taro.chooseLocation(currentOptions)
            if (res && res.latitude && res.longitude) {
                const locationData = { lat: res.latitude, lng: res.longitude }
                const fullAddress = res.address + (res.name ? ` (${res.name})` : '')

                // 调用后端逆地理编码，获取与PC端一致的结构化省市数据（short_name 如"广东省深圳市"）
                let shortName = ''
                try {
                    const geoRes: any = await GeoService.reverseGeocode(res.latitude, res.longitude)
                    const geoData = geoRes?.data || geoRes
                    shortName = geoData?.short_name || ''
                    console.log('[GeoService] 逆地理编码结果:', geoData)
                } catch (geoErr) {
                    console.warn('[GeoService] 逆地理编码失败，使用正则兜底:', geoErr)
                }

                // 兜底：后端调用失败时使用本地正则
                if (!shortName) {
                    shortName = extractLocationShortName(res.address || '', res.name || '')
                }

                if (type === 'origin') {
                    setOriginLoc(locationData)
                    setOriginShortName(shortName)
                    form.setFieldsValue({ origin_address: fullAddress })
                } else {
                    setDestLoc(locationData)
                    setDestShortName(shortName)
                    form.setFieldsValue({ destination_address: fullAddress })
                }
            }
        } catch (error) {
            console.log('取消选择位置或出错', error)
        }
    }

    const submit = async () => {
        try {
            // Use validateFields for robust validation including required checks from Form.Item rules
            const values: any = await form.validateFields()

            setLoading(true)
            const payload = {
                // 1. Route & Time
                etd: values.etd ? new Date(values.etd).toISOString() : undefined,
                eta: values.eta ? new Date(values.eta).toISOString() : undefined,

                // Use React state directly for origin/destination (form.setFieldsValue is unreliable in NutUI for readonly fields)
                origin: originShortName || (values.origin_address ? extractLocationShortName(values.origin_address, '') : ''),
                destination: destShortName || (values.destination_address ? extractLocationShortName(values.destination_address, '') : ''),
                origin_lat: originLoc && originLoc.lat ? Number(originLoc.lat) : undefined,
                origin_lng: originLoc && originLoc.lng ? Number(originLoc.lng) : undefined,
                origin_address: values.origin_address,
                dest_lat: destLoc && destLoc.lat ? Number(destLoc.lat) : undefined,
                dest_lng: destLoc && destLoc.lng ? Number(destLoc.lng) : undefined,
                dest_address: values.destination_address,

                auto_status_enabled: true,
                origin_radius: 1000,
                dest_radius: 1000,
                origin_port_code: '',
                dest_port_code: '',

                sender_name: values.sender_name,
                sender_phone: values.sender_phone,
                receiver_name: values.receiver_name,
                receiver_phone: values.receiver_phone,

                // 2. Transport Config
                // 2. Transport Config (fallback to empty string if undefined)
                transport_type: parseOptionValue(transportTypeOptions, values.transport_type) || '',
                transport_mode: parseOptionValue(transportModeOptions, values.transport_mode) || 'lcl',
                container_type: parseOptionValue(containerTypeOptions, values.container_type) || '',
                cargo_type: parseOptionValue(cargoTypeOptions, values.cargo_type) || 'general',
                device_id: values.device_id || '',

                // 3. Cargo & Docs
                cargo_name: values.cargo_name || 'General Cargo', // if user enters '自行车', it will be retained
                pieces: values.pieces ? Number(values.pieces) : undefined,
                weight: values.weight ? Number(values.weight) : undefined,
                volume: values.volume ? Number(values.volume) : undefined,

                bill_of_lading: values.bill_of_lading,
                container_no: values.container_no,
                seal_no: values.seal_no,
                vessel_name: values.vessel_name,
                voyage_no: values.voyage_no,
                carrier: values.carrier,

                po_numbers: values.po_numbers,
                sku_ids: values.sku_ids,
                fba_shipment_id: values.fba_shipment_id,

                // 4. Cost Info
                freight_cost: values.freight_cost ? Number(values.freight_cost) : undefined,
                surcharges: values.surcharges ? Number(values.surcharges) : undefined,
                customs_fee: values.customs_fee ? Number(values.customs_fee) : undefined,
                other_cost: values.other_cost ? Number(values.other_cost) : undefined,
                total_cost: (values.freight_cost ? Number(values.freight_cost) : 0) +
                            (fieldConfig.surcharges && values.surcharges ? Number(values.surcharges) : 0) +
                            (fieldConfig.customs_fee && values.customs_fee ? Number(values.customs_fee) : 0) +
                            (fieldConfig.other_cost && values.other_cost ? Number(values.other_cost) : 0),

                status: 'pending'
            }

            // Stripping undefined, NaN and extreme nulls to ensure clean JSON for Golang BindJSON
            Object.keys(payload).forEach(key => {
                if (payload[key] === undefined || Number.isNaN(payload[key])) {
                    delete payload[key]
                }
            })

            console.log('Final Request Payload:', payload)

            await ShipmentService.create(payload)
            Toast.show({ title: '创建成功', icon: 'success' })
            setTimeout(() => {
                Taro.navigateBack()
            }, 1000)
        } catch (err: any) {
            console.error('Submit error:', err)
            if (err.errorFields || (err.errors && err.errors.length > 0)) {
                // Form validation error
                Toast.show('请填写所有必填项(*)')
            } else {
                // API or other error
                Taro.showToast({ title: '创建失败: ' + (err.message || '未知错误'), icon: 'none' })
            }
        } finally {
            setLoading(false)
        }
    }

    return (
        <View className="create-container">
            <Form form={form} divider labelPosition="left">
                {/* 发货地与目的地展示 (自动根据地址获取，仅做展示) */}
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
                <Form.Item label="预计出发" name="etd" onClick={() => setShowEtd(true)}>
                    <Input placeholder="请选择ETD" value={etdDesc} readonly align="right" />
                </Form.Item>
                <Form.Item label="预计到达" name="eta" onClick={() => setShowEta(true)}>
                    <Input placeholder="请选择ETA" value={etaDesc} readonly align="right" />
                </Form.Item>
                {/* 发货方信息 */}
                <Form.Item label="* 发货人" name="sender_name" required>
                    <Input placeholder="姓名" align="right" />
                </Form.Item>
                <Form.Item label="* 发货电话" name="sender_phone" required>
                    <Input placeholder="电话" align="right" />
                </Form.Item>
                <Form.Item label="* 发货地址" name="origin_address" required onClick={() => handleChooseLocation('origin')}>
                    <Input placeholder="点击选择发货地址..." align="right" readonly />
                </Form.Item>

                {/* 间隔区 */}
                <View style={{ height: '12px', backgroundColor: '#f5f6f7', margin: '10px -16px' }}></View>

                {/* 收货方信息 */}
                <Form.Item label="* 收货人" name="receiver_name" required>
                    <Input placeholder="姓名" align="right" />
                </Form.Item>
                <Form.Item label="* 收货电话" name="receiver_phone" required>
                    <Input placeholder="电话" align="right" />
                </Form.Item>
                <Form.Item label="* 收货地址" name="destination_address" required onClick={() => handleChooseLocation('dest')}>
                    <Input placeholder="点击选择收货地址..." align="right" readonly />
                </Form.Item>

                {/* 2. 运输配置 */}
                <View className="section-title">🚚 运输配置</View>
                <Form.Item label="* 运输类型" name="transport_type" required onClick={() => setShowTransportType(true)}>
                    <Input placeholder="请选择" value={transportTypeDesc} readonly align="right" />
                </Form.Item>
                <Form.Item label="运输模式" name="transport_mode" onClick={() => setShowTransportMode(true)}>
                    <Input placeholder="请选择" value={transportModeDesc} readonly align="right" />
                </Form.Item>
                {/* Conditional rendering for Container Type if FCL */}
                {form.getFieldValue('transport_mode') === 'fcl' && (
                    <Form.Item label="柜型" name="container_type" onClick={() => setShowContainerType(true)}>
                        <Input placeholder="请选择" value={containerTypeDesc} readonly align="right" />
                    </Form.Item>
                )}
                <Form.Item label="货物类型" name="cargo_type" onClick={() => setShowCargoType(true)}>
                    <Input placeholder="请选择" value={cargoTypeDesc} readonly align="right" />
                </Form.Item>
                <Form.Item label="设备ID" name="device_id">
                    <Input placeholder="输入设备ID" align="right" />
                </Form.Item>

                {/* 3. 货物与单证 */}
                <View className="section-title">📦 货物与单证</View>
                <Form.Item label="货物名称" name="cargo_name">
                    <Input placeholder="输入货物名称" align="right" />
                </Form.Item>
                <Form.Item label="件数" name="pieces">
                    <Input placeholder="0" type="number" align="right" />
                </Form.Item>
                <Form.Item label="重量(kg)" name="weight">
                    <Input placeholder="0.0" type="number" align="right" />
                </Form.Item>
                <Form.Item label="体积(m³)" name="volume">
                    <Input placeholder="0.0" type="number" align="right" />
                </Form.Item>

                {fieldConfig.bill_of_lading && (
                    <Form.Item label="提单号" name="bill_of_lading">
                        <Input placeholder="MBL/HBL/AWB" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.container_no && (
                    <Form.Item label="箱号" name="container_no">
                        <Input placeholder="输入箱号" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.seal_no && (
                    <Form.Item label="封条号" name="seal_no">
                        <Input placeholder="输入封条号" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.vessel_name && (
                    <Form.Item label="船名" name="vessel_name">
                        <Input placeholder="输入船名" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.voyage_no && (
                    <Form.Item label="航次" name="voyage_no">
                        <Input placeholder="输入航次" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.carrier && (
                    <Form.Item label="船司/航司" name="carrier">
                        <Input placeholder="输入船司/航司" align="right" />
                    </Form.Item>
                )}

                {hasOrderFields && <View className="section-title">📋 订单关联</View>}
                {fieldConfig.po_numbers && (
                    <Form.Item label="PO单号" name="po_numbers">
                        <Input placeholder="输入PO单号" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.sku_ids && (
                    <Form.Item label="SKU ID" name="sku_ids">
                        <Input placeholder="输入SKU ID" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.fba_shipment_id && (
                    <Form.Item label="FBA编号" name="fba_shipment_id">
                        <Input placeholder="输入FBA编号" align="right" />
                    </Form.Item>
                )}

                {/* 4. 费用信息 */}
                <View className="section-title">💰 费用信息</View>
                <Form.Item label="运费" name="freight_cost">
                    <Input placeholder="0.00" type="digit" align="right" />
                </Form.Item>
                {fieldConfig.surcharges && (
                    <Form.Item label="附加费" name="surcharges">
                        <Input placeholder="0.00" type="digit" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.customs_fee && (
                    <Form.Item label="关税" name="customs_fee">
                        <Input placeholder="0.00" type="digit" align="right" />
                    </Form.Item>
                )}
                {fieldConfig.other_cost && (
                    <Form.Item label="其他费用" name="other_cost">
                        <Input placeholder="0.00" type="digit" align="right" />
                    </Form.Item>
                )}

            </Form>

            {/* Pickers */}
            <Picker
                visible={showTransportType}
                options={transportTypeOptions}
                onConfirm={(list, values) => {
                    confirmPicker('transport_type', transportTypeOptions, values, setTransportTypeDesc)
                    setShowTransportType(false)
                }}
                onClose={() => setShowTransportType(false)}
            />
            <Picker
                visible={showTransportMode}
                options={transportModeOptions}
                onConfirm={(list, values) => {
                    confirmPicker('transport_mode', transportModeOptions, values, setTransportModeDesc)
                    setShowTransportMode(false)
                }}
                onClose={() => setShowTransportMode(false)}
            />
            <Picker
                visible={showContainerType}
                options={containerTypeOptions}
                onConfirm={(list, values) => {
                    confirmPicker('container_type', containerTypeOptions, values, setContainerTypeDesc)
                    setShowContainerType(false)
                }}
                onClose={() => setShowContainerType(false)}
            />
            <Picker
                visible={showCargoType}
                options={cargoTypeOptions}
                onConfirm={(list, values) => {
                    confirmPicker('cargo_type', cargoTypeOptions, values, setCargoTypeDesc)
                    setShowCargoType(false)
                }}
                onClose={() => setShowCargoType(false)}
            />
            <DatePicker
                visible={showEtd}
                type="date"
                defaultValue={etdDesc ? new Date(etdDesc) : new Date()}
                onConfirm={(options, values) => {
                    if (values) confirmDate(values, setEtdDesc, 'etd')
                    setShowEtd(false)
                }}
                onClose={() => setShowEtd(false)}
            />
            <DatePicker
                visible={showEta}
                type="date"
                defaultValue={etaDesc ? new Date(etaDesc) : new Date()}
                onConfirm={(options, values) => {
                    if (values) confirmDate(values, setEtaDesc, 'eta')
                    setShowEta(false)
                }}
                onClose={() => setShowEta(false)}
            />

            <Button type="primary" block loading={loading} onClick={submit} style={{ marginTop: '20px', marginBottom: '40px' }}>
                提交运单
            </Button>
            <Toast />
        </View>
    )
}

export default CreateShipment
