/**
 * 坐标系转换工具
 * 
 * 中国使用的坐标系：
 * - WGS-84：GPS原始坐标，国际标准（Google Earth、OpenStreetMap、CartoDB使用）
 * - GCJ-02：火星坐标系，中国国家测绘局制定（高德、腾讯地图使用）
 * - BD-09：百度坐标系，在GCJ-02基础上再次加密（百度地图使用）
 * 
 * 本文件提供 GCJ-02 到 WGS-84 的转换，用于在国际地图上正确显示中国地图API返回的坐标
 */

// 椭球参数
const a = 6378245.0; // 长半轴
const ee = 0.00669342162296594323; // 扁率

/**
 * 判断坐标是否在中国境内
 * 只有中国境内的坐标才需要转换
 */
export function isInChina(lng: number, lat: number): boolean {
    return lng > 73.66 && lng < 135.05 && lat > 3.86 && lat < 53.55;
}

/**
 * GCJ-02 转换为 WGS-84
 * @param gcjLng GCJ-02 经度
 * @param gcjLat GCJ-02 纬度
 * @returns [WGS-84经度, WGS-84纬度]
 */
export function gcj02ToWgs84(gcjLng: number, gcjLat: number): [number, number] {
    // 境外坐标不需要转换
    if (!isInChina(gcjLng, gcjLat)) {
        return [gcjLng, gcjLat];
    }

    let dLat = transformLat(gcjLng - 105.0, gcjLat - 35.0);
    let dLng = transformLng(gcjLng - 105.0, gcjLat - 35.0);

    const radLat = (gcjLat / 180.0) * Math.PI;
    let magic = Math.sin(radLat);
    magic = 1 - ee * magic * magic;
    const sqrtMagic = Math.sqrt(magic);

    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * Math.PI);
    dLng = (dLng * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * Math.PI);

    const wgsLat = gcjLat - dLat;
    const wgsLng = gcjLng - dLng;

    return [wgsLng, wgsLat];
}

/**
 * WGS-84 转换为 GCJ-02
 * @param wgsLng WGS-84 经度
 * @param wgsLat WGS-84 纬度
 * @returns [GCJ-02经度, GCJ-02纬度]
 */
export function wgs84ToGcj02(wgsLng: number, wgsLat: number): [number, number] {
    // 境外坐标不需要转换
    if (!isInChina(wgsLng, wgsLat)) {
        return [wgsLng, wgsLat];
    }

    let dLat = transformLat(wgsLng - 105.0, wgsLat - 35.0);
    let dLng = transformLng(wgsLng - 105.0, wgsLat - 35.0);

    const radLat = (wgsLat / 180.0) * Math.PI;
    let magic = Math.sin(radLat);
    magic = 1 - ee * magic * magic;
    const sqrtMagic = Math.sqrt(magic);

    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * Math.PI);
    dLng = (dLng * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * Math.PI);

    const gcjLat = wgsLat + dLat;
    const gcjLng = wgsLng + dLng;

    return [gcjLng, gcjLat];
}

// 纬度转换辅助函数
function transformLat(x: number, y: number): number {
    let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x));
    ret += ((20.0 * Math.sin(6.0 * x * Math.PI) + 20.0 * Math.sin(2.0 * x * Math.PI)) * 2.0) / 3.0;
    ret += ((20.0 * Math.sin(y * Math.PI) + 40.0 * Math.sin((y / 3.0) * Math.PI)) * 2.0) / 3.0;
    ret += ((160.0 * Math.sin((y / 12.0) * Math.PI) + 320 * Math.sin((y * Math.PI) / 30.0)) * 2.0) / 3.0;
    return ret;
}

// 经度转换辅助函数
function transformLng(x: number, y: number): number {
    let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x));
    ret += ((20.0 * Math.sin(6.0 * x * Math.PI) + 20.0 * Math.sin(2.0 * x * Math.PI)) * 2.0) / 3.0;
    ret += ((20.0 * Math.sin(x * Math.PI) + 40.0 * Math.sin((x / 3.0) * Math.PI)) * 2.0) / 3.0;
    ret += ((150.0 * Math.sin((x / 12.0) * Math.PI) + 300.0 * Math.sin((x / 30.0) * Math.PI)) * 2.0) / 3.0;
    return ret;
}

/**
 * 批量转换坐标点
 * @param points GCJ-02 坐标点数组 [{lng, lat}, ...]
 * @returns WGS-84 坐标点数组
 */
export function batchGcj02ToWgs84<T extends { longitude: number; latitude: number }>(
    points: T[]
): T[] {
    return points.map(point => {
        const [wgsLng, wgsLat] = gcj02ToWgs84(point.longitude, point.latitude);
        return {
            ...point,
            longitude: wgsLng,
            latitude: wgsLat,
        };
    });
}
