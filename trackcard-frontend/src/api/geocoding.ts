/**
 * Nominatim Geocoding Service
 * 免费的OpenStreetMap地理编码API
 * https://nominatim.openstreetmap.org/
 */

const NOMINATIM_BASE_URL = 'https://nominatim.openstreetmap.org';

export interface GeocodingResult {
    lat: number;
    lng: number;
    displayName: string;
    address?: {
        road?: string;
        city?: string;
        state?: string;
        country?: string;
    };
}

/**
 * 地址转坐标（正向地理编码）
 * @param address 地址字符串，如 "上海市浦东新区张江高科"
 * @returns 坐标结果或null
 */
export async function geocodeAddress(address: string): Promise<GeocodingResult | null> {
    try {
        const params = new URLSearchParams({
            q: address,
            format: 'json',
            addressdetails: '1',
            limit: '1',
        });

        const response = await fetch(`${NOMINATIM_BASE_URL}/search?${params}`, {
            headers: {
                'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
                'User-Agent': 'TrackCard-Cargo-Platform/1.0',
            },
        });

        if (!response.ok) {
            throw new Error(`Geocoding failed: ${response.status}`);
        }

        const data = await response.json();

        if (data.length === 0) {
            return null;
        }

        const result = data[0];
        return {
            lat: parseFloat(result.lat),
            lng: parseFloat(result.lon),
            displayName: result.display_name,
            address: result.address,
        };
    } catch (error) {
        console.error('Geocoding error:', error);
        return null;
    }
}

/**
 * 坐标转地址（反向地理编码）
 * @param lat 纬度
 * @param lng 经度
 * @returns 地址结果或null
 */
export async function reverseGeocode(lat: number, lng: number): Promise<GeocodingResult | null> {
    try {
        const params = new URLSearchParams({
            lat: lat.toString(),
            lon: lng.toString(),
            format: 'json',
            addressdetails: '1',
        });

        const response = await fetch(`${NOMINATIM_BASE_URL}/reverse?${params}`, {
            headers: {
                'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
                'User-Agent': 'TrackCard-Cargo-Platform/1.0',
            },
        });

        if (!response.ok) {
            throw new Error(`Reverse geocoding failed: ${response.status}`);
        }

        const result = await response.json();

        if (result.error) {
            return null;
        }

        return {
            lat: parseFloat(result.lat),
            lng: parseFloat(result.lon),
            displayName: result.display_name,
            address: result.address,
        };
    } catch (error) {
        console.error('Reverse geocoding error:', error);
        return null;
    }
}

/**
 * 搜索地址（返回多个结果供选择）
 * @param query 搜索关键词
 * @param limit 返回结果数量
 * @returns 搜索结果数组
 */
export async function searchAddress(query: string, limit: number = 5): Promise<GeocodingResult[]> {
    try {
        const params = new URLSearchParams({
            q: query,
            format: 'json',
            addressdetails: '1',
            limit: limit.toString(),
        });

        const response = await fetch(`${NOMINATIM_BASE_URL}/search?${params}`, {
            headers: {
                'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
                'User-Agent': 'TrackCard-Cargo-Platform/1.0',
            },
        });

        if (!response.ok) {
            return [];
        }

        const data = await response.json();

        return data.map((item: any) => ({
            lat: parseFloat(item.lat),
            lng: parseFloat(item.lon),
            displayName: item.display_name,
            address: item.address,
        }));
    } catch (error) {
        console.error('Address search error:', error);
        return [];
    }
}
