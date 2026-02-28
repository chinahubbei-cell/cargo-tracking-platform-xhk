import React, { useState, useEffect } from 'react'
import { View, Text, ScrollView } from '@tarojs/components'
import Taro, { useDidShow } from '@tarojs/taro'
import { Button, SearchBar, Tag, Card, Toast } from '@nutui/nutui-react-taro'
import { ShipmentService } from '../../services/api'
import './index.css'

function Index() {
  const [list, setList] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')

  const fetchList = async () => {
    setLoading(true)
    try {
      // API supports params: { search: keyword }
      const res: any = await ShipmentService.list({ search: keyword, page: 1, pageSize: 20 })
      // res.data.data is the list if using standard response
      if (res.data && Array.isArray(res.data.data)) {
        setList(res.data.data)
      } else if (Array.isArray(res)) {
        setList(res)
      } else if (res.data && Array.isArray(res.data)) {
        setList(res.data)
      }
    } catch (err) {
      console.error(err)
      // Toast.fail('加载失败')
    } finally {
      setLoading(false)
    }
  }

  useDidShow(() => {
    fetchList()
  })

  // Handle Scan
  const handleScan = async () => {
    try {
      const res = await Taro.scanCode({ scanType: ['barCode', 'qrCode'] })
      console.log(res.result)
      const deviceId = res.result
      // Ideally show modal to select shipment to bind, or bind to new.
      // For MVP, lets just Toast the result or navigate to bind page.
      Toast.show({ title: `扫码结果: ${deviceId}` })
    } catch (err) {
      console.error(err)
    }
  }

  const navigateToCreate = () => {
    Taro.navigateTo({ url: '/pages/shipment/create/index' })
  }

  const navigateToDetail = (id: string) => {
    Taro.navigateTo({ url: `/pages/shipment/detail/index?id=${id}` })
    // Toast.show(`查看详情: ${id}`)
  }

  const STATUS_MAP: Record<string, string> = {
    'pending': '待发货',
    'in_transit': '运输中',
    'delivered': '已送达',
    'cancelled': '已取消'
  }

  return (
    <View className="index-page" style={{ height: '100vh', backgroundColor: '#f5f5f5' }}>
      <View style={{ padding: '10px', backgroundColor: '#fff' }}>
        <SearchBar
          placeholder="搜索运单号/客户"
          value={keyword}
          onChange={(val) => setKeyword(val)}
          onSearch={fetchList}
        />
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
      <Toast />
    </View>
  )
}

export default Index
