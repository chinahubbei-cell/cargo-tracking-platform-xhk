import { useState, useEffect } from 'react';
import api from '../api/client';

interface ShipmentFieldConfig {
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
}

// 默认配置（全部关闭）
const defaultConfig: ShipmentFieldConfig = {
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
};

/**
 * 运单字段配置Hook
 * 用于管理运单字段的显示/隐藏配置
 */
export const useShipmentFieldConfig = () => {
  const [config, setConfig] = useState<ShipmentFieldConfig>(defaultConfig);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const res = await api.getShipmentFieldConfig();
      if (res.data) {
        setConfig(res.data);
      }
    } catch (error) {
      console.error('加载运单字段配置失败:', error);
      // 保持默认配置
    } finally {
      setLoading(false);
    }
  };

  /**
   * 检查字段是否可见
   */
  const isFieldVisible = (fieldName: keyof ShipmentFieldConfig): boolean => {
    return config[fieldName] === true;
  };

  /**
   * 检查某个区域是否有可见字段
   */
  const hasVisibleFieldsInArea = (area: 'document' | 'shipping' | 'order' | 'fee') => {
    switch (area) {
      case 'document':
        return config.bill_of_lading;
      case 'shipping':
        return config.container_no || config.seal_no || config.vessel_name ||
               config.voyage_no || config.carrier;
      case 'order':
        return config.po_numbers || config.sku_ids || config.fba_shipment_id;
      case 'fee':
        return config.surcharges || config.customs_fee || config.other_cost;
      default:
        return true;
    }
  };

  /**
   * 计算费用区域的字段数量（用于自适应Col span）
   */
  const getFeeFieldCount = () => {
    let count = 1; // 运费始终显示
    if (config.surcharges) count++;
    if (config.customs_fee) count++;
    if (config.other_cost) count++;
    return count;
  };

  return {
    config,
    loading,
    isFieldVisible,
    hasVisibleFieldsInArea,
    getFeeFieldCount,
    setConfig,
    refreshConfig: loadConfig,
  };
};
