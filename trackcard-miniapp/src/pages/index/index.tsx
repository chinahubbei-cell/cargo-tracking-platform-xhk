import React, { useState } from 'react'
import { View, Text, ScrollView, Input } from '@tarojs/components'
import Taro, { useDidShow } from '@tarojs/taro'
import { Button, Tag } from '@nutui/nutui-react-taro'
import { ShipmentService } from '../../services/api'
import './index.css'

function Index() {
  const [list, setList] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')

  const fetchList = async (searchVal?: string) => {
    setLoading(true)
    try {
      const search = searchVal !== undefined ? searchVal : keyword
      const res: any = await ShipmentService.list({ search, page: 1, pageSize: 20 })
      if (res.data && Array.isArray(res.data.data)) {
        setList(res.data.data)
      } else if (Array.isArray(res)) {
        setList(res)
      } else if (res.data && Array.isArray(res.data)) {
        setList(res.data)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useDidShow(() => {
    fetchList()
  })

  const handleSearch = () => {
    fetchList(keyword)
  }

  const handleClear = () => {
    setKeyword('')
    fetchList('')
  }

  // 从二维码内容中提取设备号
  const extractDeviceIdFromQR = (qrContent: string): string => {
    const content = (qrContent || '').trim()
    if (!content) return ''
    if (content.startsWith('http://') || content.startsWith('https://')) {
      try {
        const url = new URL(content)
        const paramKeys = ['imei', 'id', 'deviceId', 'device_id', 'sn', 'devid']
        for (const key of paramKeys) {
          const val = url.searchParams.get(key)
          if (val) return val
        }
        const pathParts = url.pathname.split('/').filter(Boolean)
        if (pathParts.length > 0) {
          const lastPart = pathParts[pathParts.length - 1]
          if (/^[A-Za-z0-9_-]{4,}$/.test(lastPart)) return lastPart
        }
      } catch (e) { /* ignore */ }
    }
    if (content.startsWith('{')) {
      try {
        const obj = JSON.parse(content)
        return obj.deviceId || obj.device_id || obj.imei || obj.id || obj.sn || ''
      } catch (e) { /* ignore */ }
    }
    return content
  }

  // 扫码 → 提取设备号 → 跳转创建运单页自动填充设备号
  const handleScan = async () => {
    try {
      const res = await Taro.scanCode({ scanType: ['barCode', 'qrCode'] })
      const deviceId = extractDeviceIdFromQR(res.result)
      if (!deviceId) {
        Taro.showToast({ title: '未识别到设备号', icon: 'none' })
        return
      }
      Taro.navigateTo({ url: `/pages/shipment/create/index?deviceId=${encodeURIComponent(deviceId)}` })
    } catch (err: any) {
      if (err?.errMsg && !err.errMsg.includes('scanCode:fail cancel')) {
        console.error('[Scan] 扫码失败:', err)
      }
    }
  }

  const navigateToCreate = () => {
    Taro.navigateTo({ url: '/pages/shipment/create/index' })
  }

  const navigateToDetail = (id: string) => {
    Taro.navigateTo({ url: `/pages/shipment/detail/index?id=${id}` })
  }

  const STATUS_MAP: Record<string, string> = {
    'pending': '待发货',
    'in_transit': '运输中',
    'delivered': '已送达',
    'cancelled': '已取消'
  }
  return (
    <View className="index-page" style={{ height: '100vh', backgroundColor: '#f5f5f5' }}>
      <View style={{ padding: '10px', backgroundColor: '#fff', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <View style={{ flex: 1, backgroundColor: '#f5f5f5', borderRadius: '20px', padding: '6px 14px', display: 'flex', alignItems: 'center' }}>
          <Input
            placeholder="搜索运单号/客户名称"
            value={keyword}
            onInput={(e) => setKeyword(e.detail.value)}
            onConfirm={() => handleSearch()}
            confirmType="search"
            style={{ flex: 1, fontSize: '14px', backgroundColor: 'transparent' }}
          />
          {keyword && (
            <Text onClick={handleClear} style={{ color: '#999', fontSize: '18px', padding: '0 4px' }}>✕</Text>
          )}
        </View>
        <Button type="primary" size="small" onClick={handleSearch} loading={loading}
          style={{ borderRadius: '20px', padding: '0 16px', height: '36px', whiteSpace: 'nowrap' }}
        >查询</Button>
      </View>

      <ScrollView scrollY style={{ height: 'calc(100vh - 120px)' }}>
        <View style={{ padding: '10px' }}>
          {list.map(item => (
            <View
              key={item.id}
              style={{ backgroundColor: '#fff', borderRadius: '8px', padding: '15px', marginBottom: '10px' }}
              onClick={() => navigateToDetail(item.id)}
            >
              <View style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
                <Text style={{ fontWeight: 'bold', fontSize: '16px' }}>{item.tracking_number || item.id || '(无单号)'}</Text>
                <Tag type={item.status === 'in_transit' ? 'success' : 'warning'}>
                  {STATUS_MAP[item.status] || item.status}
                </Tag>
              </View>
              <View style={{ fontSize: '14px', color: '#666' }}>
                <View>客户: {item.receiver_name || item.sender_name || '-'}</View>
                <View>发货地: {item.origin || (item.origin_address ? item.origin_address.split(' ')[0] : '-')}</View>
                <View>目的地: {item.destination || (item.dest_address ? item.dest_address.split(' ')[0] : '-')}</View>
              </View>
            </View>
          ))}
          {list.length === 0 && !loading && (
            <View style={{ textAlign: 'center', padding: '20px', color: '#999' }}>暂无运单</View>
          )}
        </View>
      </ScrollView>

      <View style={{
        position: 'fixed',
        bottom: '20px',
        left: '0',
        right: '0',
        display: 'flex',
        justifyContent: 'space-around',
        padding: '0 20px'
      }}>
        <Button type="info" onClick={navigateToCreate} style={{ width: '45%' }}>创建运单</Button>
        <Button type="primary" onClick={handleScan} style={{ width: '45%' }}>扫码绑定</Button>
      </View>

    </View>
  )
}

export default Index
