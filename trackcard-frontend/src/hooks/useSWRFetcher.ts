/**
 * SWR 数据请求配置和通用 fetcher
 * 用于缓存API响应，避免重复请求，提升数据加载速度
 */
import axios from 'axios';

// 通用 fetcher 函数
export const fetcher = async (url: string) => {
    const token = localStorage.getItem('token');
    const res = await axios.get(url, {
        headers: { Authorization: `Bearer ${token}` }
    });
    if (res.data.success || res.data.data) {
        return res.data;
    }
    throw new Error(res.data.message || '请求失败');
};

// SWR 全局配置选项
export const swrOptions = {
    // 窗口聚焦时不自动刷新（避免频繁请求）
    revalidateOnFocus: false,
    // 网络恢复时不自动刷新
    revalidateOnReconnect: false,
    // 60秒内重复请求使用缓存
    dedupingInterval: 60000,
    // 错误时重试3次
    errorRetryCount: 3,
    // 数据保持新鲜时间：5分钟内不自动重新验证
    refreshInterval: 0,
    // 超时设置
    loadingTimeout: 10000,
};

// 静态数据配置（港口、机场、区域等较少变化的数据）
export const staticDataOptions = {
    ...swrOptions,
    // 静态数据缓存更长时间
    dedupingInterval: 300000, // 5分钟
    // 禁用自动刷新
    revalidateIfStale: false,
};

// 动态数据配置（运单、设备状态等需要及时更新的数据）
export const dynamicDataOptions = {
    ...swrOptions,
    // 动态数据缓存较短时间
    dedupingInterval: 30000, // 30秒
};
