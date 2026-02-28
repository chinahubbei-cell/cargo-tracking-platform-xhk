/**
 * 机场数据 SWR Hooks
 * 提供缓存的机场数据请求
 */
import useSWR from 'swr';
import { fetcher, staticDataOptions } from './useSWRFetcher';

// 机场列表（支持分页和筛选）
export function useAirports(params?: {
    page?: number;
    pageSize?: number;
    search?: string;
    region?: string;
    tier?: number;
}) {
    const queryString = params
        ? new URLSearchParams({
            ...(params.page && { page: params.page.toString() }),
            ...(params.pageSize && { page_size: params.pageSize.toString() }),
            ...(params.search && { search: params.search }),
            ...(params.region && { region: params.region }),
            ...(params.tier && { tier: params.tier.toString() }),
        }).toString()
        : '';

    const url = queryString ? `/api/airports?${queryString}` : '/api/airports';

    return useSWR(url, fetcher, staticDataOptions);
}

// 地图用机场数据（获取前200个用于地图显示）
export function useMapAirports() {
    return useSWR('/api/airports?page_size=200', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 300000, // 5分钟缓存
    });
}

// 机场围栏数据
export function useAirportGeofences() {
    return useSWR('/api/airport-geofences', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 300000, // 5分钟缓存
    });
}

// 区域列表
export function useAirportRegions() {
    return useSWR('/api/airports/regions', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 3600000, // 1小时缓存
    });
}

// 机场距离计算
export function useAirportDistance(from: string | null, to: string | null) {
    const shouldFetch = from && to;
    const url = shouldFetch ? `/api/airports/distance?from=${from}&to=${to}` : null;

    return useSWR(url, fetcher, {
        ...staticDataOptions,
        dedupingInterval: 60000, // 1分钟缓存
    });
}
