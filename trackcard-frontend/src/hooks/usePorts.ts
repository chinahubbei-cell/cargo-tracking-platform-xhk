/**
 * 港口数据 SWR Hooks
 * 提供缓存的港口数据请求
 */
import useSWR from 'swr';
import { fetcher, staticDataOptions } from './useSWRFetcher';

// 港口列表
export function usePorts(params?: { page?: number; pageSize?: number; search?: string; region?: string }) {
    const queryString = params
        ? new URLSearchParams({
            ...(params.page && { page: params.page.toString() }),
            ...(params.pageSize && { page_size: params.pageSize.toString() }),
            ...(params.search && { search: params.search }),
            ...(params.region && { region: params.region }),
        }).toString()
        : '';

    const url = queryString ? `/api/ports?${queryString}` : '/api/ports';

    return useSWR(url, fetcher, staticDataOptions);
}

// 港口围栏数据（静态数据，长缓存）
export function usePortGeofences() {
    return useSWR('/api/port-geofences', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 300000, // 5分钟缓存
    });
}

// 港口区域列表（静态数据）
export function usePortRegions() {
    return useSWR('/api/ports/regions', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 3600000, // 1小时缓存
    });
}

// 港口航线
export function usePortRoutes() {
    return useSWR('/api/ports/shipping-lines', fetcher, {
        ...staticDataOptions,
        dedupingInterval: 600000, // 10分钟缓存
    });
}
