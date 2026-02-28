import React, { useState, useMemo, useCallback, useRef } from 'react';
import { AutoComplete, Input, Spin } from 'antd';
import { EnvironmentOutlined, LoadingOutlined } from '@ant-design/icons';
import { api } from '../api/client';

// 城市数据：包含中文名、英文名、拼音、缩写、经纬度
interface CityData {
    name: string;           // 中文名
    nameEn: string;         // 英文名
    pinyin: string;         // 拼音
    abbr: string;           // 缩写 (如 sz, bj, sh)
    country: string;        // 国家中文名
    countryCode: string;    // 国家代码
    lat: number;            // 纬度
    lng: number;            // 经度
}

// 地理编码结果
interface GeocodeResult {
    lat: number;
    lng: number;
    province: string;
    city: string;
    district: string;
    address: string;
    formatted: string;
}

const CITY_DATABASE: CityData[] = [
    // 中国主要城市
    { name: '深圳', nameEn: 'Shenzhen', pinyin: 'shenzhen', abbr: 'sz', country: '中国', countryCode: 'CN', lat: 22.5431, lng: 114.0579 },
    { name: '广州', nameEn: 'Guangzhou', pinyin: 'guangzhou', abbr: 'gz', country: '中国', countryCode: 'CN', lat: 23.1291, lng: 113.2644 },
    { name: '上海', nameEn: 'Shanghai', pinyin: 'shanghai', abbr: 'sh', country: '中国', countryCode: 'CN', lat: 31.2304, lng: 121.4737 },
    { name: '北京', nameEn: 'Beijing', pinyin: 'beijing', abbr: 'bj', country: '中国', countryCode: 'CN', lat: 39.9042, lng: 116.4074 },
    { name: '杭州', nameEn: 'Hangzhou', pinyin: 'hangzhou', abbr: 'hz', country: '中国', countryCode: 'CN', lat: 30.2741, lng: 120.1551 },
    { name: '宁波', nameEn: 'Ningbo', pinyin: 'ningbo', abbr: 'nb', country: '中国', countryCode: 'CN', lat: 29.8683, lng: 121.5440 },
    { name: '东莞', nameEn: 'Dongguan', pinyin: 'dongguan', abbr: 'dg', country: '中国', countryCode: 'CN', lat: 23.0430, lng: 113.7633 },
    { name: '天津', nameEn: 'Tianjin', pinyin: 'tianjin', abbr: 'tj', country: '中国', countryCode: 'CN', lat: 39.3434, lng: 117.3616 },
    { name: '青岛', nameEn: 'Qingdao', pinyin: 'qingdao', abbr: 'qd', country: '中国', countryCode: 'CN', lat: 36.0671, lng: 120.3826 },
    { name: '厦门', nameEn: 'Xiamen', pinyin: 'xiamen', abbr: 'xm', country: '中国', countryCode: 'CN', lat: 24.4798, lng: 118.0894 },
    { name: '苏州', nameEn: 'Suzhou', pinyin: 'suzhou', abbr: 'sz', country: '中国', countryCode: 'CN', lat: 31.2990, lng: 120.5853 },
    { name: '成都', nameEn: 'Chengdu', pinyin: 'chengdu', abbr: 'cd', country: '中国', countryCode: 'CN', lat: 30.5728, lng: 104.0668 },
    { name: '重庆', nameEn: 'Chongqing', pinyin: 'chongqing', abbr: 'cq', country: '中国', countryCode: 'CN', lat: 29.4316, lng: 106.9123 },
    { name: '武汉', nameEn: 'Wuhan', pinyin: 'wuhan', abbr: 'wh', country: '中国', countryCode: 'CN', lat: 30.5928, lng: 114.3055 },
    { name: '香港', nameEn: 'Hong Kong', pinyin: 'xianggang', abbr: 'hk', country: '中国', countryCode: 'HK', lat: 22.3193, lng: 114.1694 },

    // 美国主要城市
    { name: '洛杉矶', nameEn: 'Los Angeles', pinyin: 'luoshanji', abbr: 'la', country: '美国', countryCode: 'US', lat: 34.0522, lng: -118.2437 },
    { name: '纽约', nameEn: 'New York', pinyin: 'niuyue', abbr: 'ny', country: '美国', countryCode: 'US', lat: 40.7128, lng: -74.0060 },
    { name: '芝加哥', nameEn: 'Chicago', pinyin: 'zhijiage', abbr: 'chi', country: '美国', countryCode: 'US', lat: 41.8781, lng: -87.6298 },
    { name: '旧金山', nameEn: 'San Francisco', pinyin: 'jiujinshan', abbr: 'sf', country: '美国', countryCode: 'US', lat: 37.7749, lng: -122.4194 },
    { name: '西雅图', nameEn: 'Seattle', pinyin: 'xiyatu', abbr: 'sea', country: '美国', countryCode: 'US', lat: 47.6062, lng: -122.3321 },

    // 欧洲主要城市
    { name: '伦敦', nameEn: 'London', pinyin: 'lundun', abbr: 'ldn', country: '英国', countryCode: 'GB', lat: 51.5074, lng: -0.1278 },
    { name: '汉堡', nameEn: 'Hamburg', pinyin: 'hanbao', abbr: 'ham', country: '德国', countryCode: 'DE', lat: 53.5511, lng: 9.9937 },
    { name: '鹿特丹', nameEn: 'Rotterdam', pinyin: 'lutedan', abbr: 'rtm', country: '荷兰', countryCode: 'NL', lat: 51.9244, lng: 4.4777 },

    // 亚洲主要城市
    { name: '东京', nameEn: 'Tokyo', pinyin: 'dongjing', abbr: 'tyo', country: '日本', countryCode: 'JP', lat: 35.6762, lng: 139.6503 },
    { name: '新加坡', nameEn: 'Singapore', pinyin: 'xinjiapo', abbr: 'sin', country: '新加坡', countryCode: 'SG', lat: 1.3521, lng: 103.8198 },
    { name: '曼谷', nameEn: 'Bangkok', pinyin: 'mangu', abbr: 'bkk', country: '泰国', countryCode: 'TH', lat: 13.7563, lng: 100.5018 },
    { name: '迪拜', nameEn: 'Dubai', pinyin: 'dibai', abbr: 'dxb', country: '阿联酋', countryCode: 'AE', lat: 25.2048, lng: 55.2708 },

    // 澳洲主要城市
    { name: '悉尼', nameEn: 'Sydney', pinyin: 'xini', abbr: 'syd', country: '澳大利亚', countryCode: 'AU', lat: -33.8688, lng: 151.2093 },
];

// 搜索匹配逻辑
const searchCities = (keyword: string): CityData[] => {
    if (!keyword || keyword.length < 1) return [];

    const lowerKeyword = keyword.toLowerCase().trim();

    return CITY_DATABASE.filter(city => {
        if (city.name.includes(keyword)) return true;
        if (city.nameEn.toLowerCase().includes(lowerKeyword)) return true;
        if (city.pinyin.toLowerCase().startsWith(lowerKeyword)) return true;
        if (city.abbr.toLowerCase() === lowerKeyword) return true;
        if (city.country.includes(keyword)) return true;
        if (city.countryCode.toLowerCase() === lowerKeyword) return true;
        return false;
    }).slice(0, 10);
};

interface AddressAutocompleteProps {
    value?: string;
    onChange?: (value: string, coords?: { lat: number; lng: number }) => void;
    onLocationSelect?: (location: { lat: number; lng: number } | null) => void;
    onGeocodeResult?: (result: GeocodeResult | null) => void;  // 新增：返回完整地理编码结果
    placeholder?: string;
    style?: React.CSSProperties;
}

const AddressAutocomplete: React.FC<AddressAutocompleteProps> = ({
    value = '',
    onChange,
    onLocationSelect,
    onGeocodeResult,
    placeholder = '输入城市或详细地址...',
    style,
}) => {
    const [inputValue, setInputValue] = useState(value);
    const [options, setOptions] = useState<{ value: string; label: React.ReactNode; city?: CityData; geocode?: GeocodeResult }[]>([]);
    const [loading, setLoading] = useState(false);
    const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

    // 调用后端地理编码API
    const fetchGeocode = useCallback(async (address: string) => {
        if (address.length < 4) return null;  // 至少4个字符才调用API

        try {
            setLoading(true);
            const result = await api.geocode(address);
            return result;
        } catch (error) {
            console.error('Geocode failed:', error);
            return null;
        } finally {
            setLoading(false);
        }
    }, []);

    // 搜索处理 (带防抖)
    const handleSearch = useCallback((searchText: string) => {
        setInputValue(searchText);

        if (!searchText) {
            setOptions([]);
            return;
        }

        // 清除之前的定时器
        if (debounceTimer.current) {
            clearTimeout(debounceTimer.current);
        }

        // 先显示本地城市匹配结果
        const localResults = searchCities(searchText);
        const localOptions = localResults.map(city => ({
            value: `${city.name}, ${city.country}`,
            city,
            label: (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div>
                        <EnvironmentOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                        <strong>{city.name}</strong>
                        <span style={{ color: '#888', marginLeft: 8 }}>{city.nameEn}</span>
                    </div>
                    <span style={{ color: '#999', fontSize: 12 }}>{city.country}</span>
                </div>
            ),
        }));

        setOptions(localOptions);

        // 如果输入够长，延迟调用后端API
        if (searchText.length >= 4) {
            debounceTimer.current = setTimeout(async () => {
                const geocodeResult = await fetchGeocode(searchText);
                // 检查返回结果的类型是否包含必要属性
                if (geocodeResult) {
                    const result = geocodeResult as any;
                    const lat = result.lat || result.data?.lat;
                    const lng = result.lng || result.data?.lng;
                    const formatted = result.formatted || result.data?.address || searchText;

                    if (lat && lng) {
                        // 构造标准的 GeocodeResult
                        const normalizedGeocode: GeocodeResult = {
                            lat,
                            lng,
                            province: result.province || result.data?.province || '',
                            city: result.city || result.data?.city || '',
                            district: result.district || result.data?.district || '',
                            address: result.address || result.data?.address || searchText,
                            formatted,
                        };

                        // 在列表顶部添加地理编码结果
                        const geocodeOption = {
                            value: formatted,
                            geocode: normalizedGeocode,
                            label: (
                                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: '#f0f9ff', padding: '4px 8px', margin: '-4px -12px', borderRadius: 4 }}>
                                    <div>
                                        <EnvironmentOutlined style={{ marginRight: 8, color: '#52c41a' }} />
                                        <strong style={{ color: '#1890ff' }}>{formatted}</strong>
                                    </div>
                                    <span style={{ color: '#52c41a', fontSize: 12 }}>📍 精确定位</span>
                                </div>
                            ),
                        };
                        setOptions(prev => [geocodeOption, ...prev.filter(o => !o.geocode)]);
                    }
                }
            }, 300);  // 300ms 防抖
        }
    }, [fetchGeocode]);

    // 选择处理
    const handleSelect = useCallback((selectedValue: string, option: any) => {
        setInputValue(selectedValue);

        // 优先使用地理编码结果
        if (option?.geocode) {
            const coords = { lat: option.geocode.lat, lng: option.geocode.lng };
            if (onChange) {
                onChange(selectedValue, coords);
            }
            if (onLocationSelect) {
                onLocationSelect(coords);
            }
            if (onGeocodeResult) {
                onGeocodeResult(option.geocode);
            }
        } else if (option?.city) {
            const coords = { lat: option.city.lat, lng: option.city.lng };
            if (onChange) {
                onChange(selectedValue, coords);
            }
            if (onLocationSelect) {
                onLocationSelect(coords);
            }
            if (onGeocodeResult) {
                // 构造兼容的地理编码结果
                onGeocodeResult({
                    lat: option.city.lat,
                    lng: option.city.lng,
                    province: '',
                    city: option.city.name,
                    district: '',
                    address: selectedValue,
                    formatted: selectedValue,
                });
            }
        }
    }, [onChange, onLocationSelect, onGeocodeResult]);

    // 输入变化处理
    const handleChange = useCallback((newValue: string) => {
        setInputValue(newValue);
        if (onChange) {
            onChange(newValue);
        }
    }, [onChange]);

    // 同步外部value变化
    useMemo(() => {
        if (value !== inputValue) {
            setInputValue(value);
        }
    }, [value]);

    return (
        <AutoComplete
            value={inputValue}
            options={options}
            onSearch={handleSearch}
            onSelect={handleSelect}
            onChange={handleChange}
            style={{ width: '100%', ...style }}
            placeholder={placeholder}
        >
            <Input
                prefix={loading ? <LoadingOutlined style={{ color: '#1890ff' }} /> : <EnvironmentOutlined style={{ color: '#bfbfbf' }} />}
                allowClear
                suffix={loading ? <Spin size="small" /> : null}
            />
        </AutoComplete>
    );
};

// 导出城市数据库供其他组件使用
export { CITY_DATABASE, searchCities };
export type { CityData, GeocodeResult };
export default AddressAutocomplete;
