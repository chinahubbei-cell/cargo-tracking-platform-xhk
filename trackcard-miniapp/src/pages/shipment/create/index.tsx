import React, { useEffect, useState } from 'react'
import { View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { Button, Input, Form, Toast, Picker, DatePicker } from '@nutui/nutui-react-taro'
import { ConfigService, ShipmentFieldConfig, ShipmentService } from '../../../services/api'
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

    const confirmPicker = (options: any[], value: any[], descSetter: any) => {
        const option = options.find(o => o.value === value[0])
        descSetter(option ? option.text : '')
    }

    const confirmDate = (values: any[], descSetter: any, fieldName: string) => {
        const dateStr = values.join('-')
        descSetter(dateStr)
        form.setFieldsValue({ [fieldName]: dateStr })
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

                // Prioritize explicit origin input, then try to extract city from address, finally fallback to first part of address
                origin: values.origin || (values.origin_address ? values.origin_address.split(' ')[0] : ''),
                destination: values.destination || (values.destination_address ? values.destination_address.split(' ')[0] : ''),
                origin_address: values.origin_address,
                dest_address: values.destination_address,

                sender_name: values.sender_name,
                sender_phone: values.sender_phone,
                receiver_name: values.receiver_name,
                receiver_phone: values.receiver_phone,

                // 2. Transport Config
                transport_type: values.transport_type,
                transport_mode: values.transport_mode || 'lcl',
                container_type: values.container_type,
                cargo_type: values.cargo_type || 'general',
                device_id: values.device_id,

                // 3. Cargo & Docs
                cargo_name: values.cargo_name || 'General Cargo',
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

                status: 'pending'
            }

            console.log('Creating shipment with payload:', payload)
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
                {/* 1. 路线与时效 */}
                <View className="section-title">📍 路线与时效</View>
                <Form.Item label="预计出发" name="etd" onClick={() => setShowEtd(true)}>
                    <Input placeholder="请选择ETD" value={etdDesc} readonly align="right" />
                </Form.Item>
                <Form.Item label="预计到达" name="eta" onClick={() => setShowEta(true)}>
                    <Input placeholder="请选择ETA" value={etaDesc} readonly align="right" />
                </Form.Item>
                <Form.Item label="* 发货城市" name="origin" required>
                    <Input placeholder="城市名/港口代码" align="right" />
                </Form.Item>
                <Form.Item label="* 发货地址" name="origin_address" required>
                    <Input placeholder="详细地址" align="right" />
                </Form.Item>
                <Form.Item label="* 目的城市" name="destination" required>
                    <Input placeholder="城市名/港口代码" align="right" />
                </Form.Item>
                <Form.Item label="* 收货地址" name="destination_address" required>
                    <Input placeholder="详细地址" align="right" />
                </Form.Item>

                <Form.Item label="* 发货人" name="sender_name" required>
                    <Input placeholder="姓名" align="right" />
                </Form.Item>
                <Form.Item label="* 发货电话" name="sender_phone" required>
                    <Input placeholder="电话" align="right" />
                </Form.Item>
                <Form.Item label="* 收货人" name="receiver_name" required>
                    <Input placeholder="姓名" align="right" />
                </Form.Item>
                <Form.Item label="* 收货电话" name="receiver_phone" required>
                    <Input placeholder="电话" align="right" />
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
                    confirmPicker(transportTypeOptions, values, setTransportTypeDesc)
                    form.setFieldsValue({ transport_type: values[0] })
                    setShowTransportType(false)
                }}
                onClose={() => setShowTransportType(false)}
            />
            <Picker
                visible={showTransportMode}
                options={transportModeOptions}
                onConfirm={(list, values) => {
                    confirmPicker(transportModeOptions, values, setTransportModeDesc)
                    form.setFieldsValue({ transport_mode: values[0] })
                    setShowTransportMode(false)
                }}
                onClose={() => setShowTransportMode(false)}
            />
            <Picker
                visible={showContainerType}
                options={containerTypeOptions}
                onConfirm={(list, values) => {
                    confirmPicker(containerTypeOptions, values, setContainerTypeDesc)
                    form.setFieldsValue({ container_type: values[0] })
                    setShowContainerType(false)
                }}
                onClose={() => setShowContainerType(false)}
            />
            <Picker
                visible={showCargoType}
                options={cargoTypeOptions}
                onConfirm={(list, values) => {
                    confirmPicker(cargoTypeOptions, values, setCargoTypeDesc)
                    form.setFieldsValue({ cargo_type: values[0] })
                    setShowCargoType(false)
                }}
                onClose={() => setShowCargoType(false)}
            />
            <DatePicker
                visible={showEtd}
                type="date"
                onConfirm={(options, values) => {
                    if (values) confirmDate(values, setEtdDesc, 'etd')
                    setShowEtd(false)
                }}
                onClose={() => setShowEtd(false)}
            />
            <DatePicker
                visible={showEta}
                type="date"
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
