import React, { useState, useCallback, useRef } from 'react';
import { AutoComplete, Input, Space, Typography, Spin, message } from 'antd';
import { EnvironmentOutlined, GlobalOutlined, LoadingOutlined } from '@ant-design/icons';
import debounce from 'lodash/debounce';
import api from '../api/client';

const { Text } = Typography;

export interface AddressData {
    address: string;      // 完整地址
    shortName: string;    // 简称(省市区 或 州/郡/市)
    lat: number;          // 纬度
    lng: number;          // 经度
    province: string;     // 省
    city: string;         // 市
    district: string;     // 区
    country?: string;     // 国家
    isOversea: boolean;   // 是否海外
}

interface AddressInputProps {
    value?: string;
    onChange?: (value: string) => void;
    onAddressSelect?: (data: AddressData) => void;
    placeholder?: string;
    disabled?: boolean;
    style?: React.CSSProperties;
}

const AddressInput: React.FC<AddressInputProps> = ({
    value,
    onChange,
    onAddressSelect,
    placeholder = '请输入地址',
    disabled = false,
    style
}) => {
    const [loading, setLoading] = useState(false);
    const [options, setOptions] = useState<Array<{
        value: string;
        label: React.ReactNode;
        data: any;
    }>>([]);
    const [geocoding, setGeocoding] = useState(false);
    // 标记地址是否已经从选择中解析（避免blur时重复调用geocode）
    const addressResolvedRef = useRef(false);

    // 判断是否为中文地址（包含中文字符）
    const isChineseAddress = (text: string): boolean => {
        return /[\u4e00-\u9fa5]/.test(text);
    };

    // 防抖函数 - 输入联想
    const debouncedSearch = useCallback(
        debounce(async (keyword: string) => {
            if (!keyword || keyword.length < 2) {
                setOptions([]);
                return;
            }

            setLoading(true);
            // 用户正在输入新地址，清除已解析标记
            addressResolvedRef.current = false;
            try {
                // 自动检测是否为海外地址
                const isOversea = !isChineseAddress(keyword);
                const result = await api.addressSuggestion(keyword, undefined, isOversea);
                if (result.success && result.data && result.data.length > 0) {
                    const newOptions = result.data.map((item: any) => ({
                        value: item.title,
                        label: (
                            <Space direction="vertical" size={0} style={{ width: '100%' }}>
                                <Space>
                                    <EnvironmentOutlined style={{ color: '#1890ff' }} />
                                    <Text strong>{item.title}</Text>
                                </Space>
                                <Text type="secondary" style={{ fontSize: 12, marginLeft: 22 }}>
                                    {item.address || `${item.province}${item.city}${item.district}`}
                                </Text>
                            </Space>
                        ),
                        data: item
                    }));
                    setOptions(newOptions);
                } else {
                    setOptions([]);
                }
            } catch (err) {
                console.error('地址联想失败:', err);
                setOptions([]);
            } finally {
                setLoading(false);
            }
        }, 300),
        []
    );

    // 处理输入变化
    const handleSearch = (searchText: string) => {
        onChange?.(searchText);
        debouncedSearch(searchText);
    };

    // 处理选择
    const handleSelect = async (_value: string, option: any) => {
        const selectedData = option.data;

        if (selectedData && selectedData.lat !== undefined && selectedData.lng !== undefined) {
            // 选中的数据已有坐标，标记为已解析
            addressResolvedRef.current = true;
            const isOversea = !isChineseAddress(selectedData.title || '');

            // 根据是否海外地址生成简称
            let shortName = '';
            if (isOversea) {
                // 海外地址：使用地址标题或city作为简称
                shortName = selectedData.city || selectedData.title || '';
            } else {
                // 国内地址：省+市（只取二级，不含区县）
                shortName = `${selectedData.province || ''}${selectedData.city || ''}`;
            }

            // 优先使用完整地址，如果没有则使用 title
            const displayAddress = selectedData.address || `${selectedData.province || ''}${selectedData.city || ''}${selectedData.district || ''}${selectedData.title || ''}`;

            const addressData: AddressData = {
                address: displayAddress,
                shortName: shortName,
                lat: selectedData.lat,
                lng: selectedData.lng,
                province: selectedData.province || '',
                city: selectedData.city || '',
                district: selectedData.district || '',
                country: selectedData.nation || '', // Map nation from suggestion
                isOversea: isOversea
            };
            // 显示完整地址而非标题
            onChange?.(displayAddress);
            onAddressSelect?.(addressData);
            message.success(`已选择: ${selectedData.title}`);
        } else {
            // 需要调用geocode获取坐标
            await geocodeAddress(_value);
        }
    };

    // 地址解析
    const geocodeAddress = async (address: string) => {
        setGeocoding(true);
        try {
            // 自动检测是否为海外地址
            const isOversea = !isChineseAddress(address);
            const result = await api.geocode(address, isOversea);
            if (result.success && result.data) {
                addressResolvedRef.current = true;
                const addressData: AddressData = {
                    address: result.data.address,
                    shortName: result.data.short_name,
                    lat: result.data.lat,
                    lng: result.data.lng,
                    province: result.data.province,
                    city: result.data.city,
                    district: result.data.district,
                    country: result.data.nation, // Map nation to country
                    isOversea: result.data.is_oversea
                };
                onChange?.(address);
                onAddressSelect?.(addressData);
                message.success(`已解析: ${addressData.shortName || address}`);
            } else {
                // 解析失败时，对于海外地址仍保存地址文本
                if (isOversea) {
                    addressResolvedRef.current = true;
                    const addressData: AddressData = {
                        address: address,
                        shortName: address,
                        lat: 0,
                        lng: 0,
                        province: '',
                        city: '',
                        district: '',
                        country: '',
                        isOversea: true
                    };
                    onChange?.(address);
                    onAddressSelect?.(addressData);
                    message.warning('海外地址无坐标，建议从下拉列表选择');
                } else {
                    message.warning('地址解析失败，请从下拉列表选择');
                }
            }
        } catch (err) {
            console.error('地址解析失败:', err);
            // 错误时也对海外地址做容错处理
            const isOversea = !isChineseAddress(address);
            if (isOversea) {
                addressResolvedRef.current = true;
                const addressData: AddressData = {
                    address: address,
                    shortName: address,
                    lat: 0,
                    lng: 0,
                    province: '',
                    city: '',
                    district: '',
                    country: '',
                    isOversea: true
                };
                onChange?.(address);
                onAddressSelect?.(addressData);
                message.warning('海外地址无坐标，建议从下拉列表选择');
            } else {
                message.warning('地址解析失败');
            }
        } finally {
            setGeocoding(false);
        }
    };

    // 处理失去焦点时自动解析地址
    const handleBlur = async () => {
        // 如果地址已经从选择中解析，不再重复调用geocode
        if (addressResolvedRef.current) {
            return;
        }
        if (value && value.length >= 2 && onAddressSelect) {
            await geocodeAddress(value);
        }
    };

    return (
        <AutoComplete
            value={value}
            options={options}
            onSearch={handleSearch}
            onSelect={handleSelect}
            onBlur={handleBlur}
            disabled={disabled}
            style={{ width: '100%', ...style }}
            notFoundContent={loading ? <Spin size="small" /> : null}
        >
            <Input
                placeholder={placeholder}
                prefix={<GlobalOutlined style={{ color: '#999' }} />}
                suffix={geocoding ? <LoadingOutlined spin style={{ color: '#1890ff' }} /> : null}
            />
        </AutoComplete>
    );
};

export default AddressInput;
