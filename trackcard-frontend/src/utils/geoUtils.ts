/**
 * 地理计算工具函数
 * 使用 Haversine 公式计算大圆距离
 */

/**
 * 角度转弧度
 */
function deg2rad(deg: number): number {
    return deg * (Math.PI / 180);
}

/**
 * 计算两个经纬度坐标之间的大圆距离 (Haversine Formula)
 * @param lat1 起始纬度
 * @param lon1 起始经度
 * @param lat2 目的纬度
 * @param lon2 目的经度
 * @returns 距离 (公里 km)
 */
export function getDistanceFromLatLonInKm(
    lat1: number,
    lon1: number,
    lat2: number,
    lon2: number
): number {
    const R = 6371; // 地球半径 (km)
    const dLat = deg2rad(lat2 - lat1);
    const dLon = deg2rad(lon2 - lon1);

    const a =
        Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos(deg2rad(lat1)) *
        Math.cos(deg2rad(lat2)) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2);

    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    const d = R * c; // Distance in km
    return parseFloat(d.toFixed(2));
}

/**
 * 计算距离（返回公里和海里）
 */
export function calculateDistances(
    lat1: number,
    lon1: number,
    lat2: number,
    lon2: number
): { km: number; nm: number } {
    const km = getDistanceFromLatLonInKm(lat1, lon1, lat2, lon2);
    const nm = parseFloat((km / 1.852).toFixed(2)); // 1海里 = 1.852公里
    return { km, nm };
}

/**
 * 估算飞行时间
 * @param distanceKm 距离(公里) - 大圆距离
 * @param cruiseSpeedKmh 巡航速度(公里/小时)，默认850
 * @param routingFactor 航路修正系数(1.05-1.10)，默认1.08
 *   - 实际航程比大圆距离长5-10%（避开禁飞区、顺风逆风绕行、空中排队等）
 * @returns 预计飞行时间对象
 */
export function estimateFlightTime(
    distanceKm: number,
    cruiseSpeedKmh: number = 850,
    routingFactor: number = 1.08
): { hours: number; formatted: string; actualDistanceKm: number } {
    // 应用航路修正系数
    const actualDistanceKm = distanceKm * routingFactor;
    const hours = actualDistanceKm / cruiseSpeedKmh;
    const h = Math.floor(hours);
    const m = Math.round((hours - h) * 60);

    let formatted: string;
    if (h > 0) {
        formatted = `${h}小时${m}分钟`;
    } else {
        formatted = `${m}分钟`;
    }

    return {
        hours: parseFloat(hours.toFixed(2)),
        formatted,
        actualDistanceKm: parseFloat(actualDistanceKm.toFixed(2)),
    };
}

/**
 * 估算海运时间
 * @param distanceNm 距离(海里)
 * @param speedKnots 航速(节)，默认18节
 * @returns 预计航行时间对象
 */
export function estimateSeaTime(
    distanceNm: number,
    speedKnots: number = 18
): { hours: number; days: number; formatted: string } {
    const hours = distanceNm / speedKnots;
    const days = hours / 24;
    const d = Math.floor(days);
    const remainingHours = Math.round((days - d) * 24);

    let formatted: string;
    if (d > 0) {
        formatted = `${d}天${remainingHours}小时`;
    } else {
        formatted = `${remainingHours}小时`;
    }

    return {
        hours: parseFloat(hours.toFixed(2)),
        days: parseFloat(days.toFixed(2)),
        formatted,
    };
}

/**
 * 计算机场间距离和预计飞行时间
 */
export function calculateAirportDistance(
    fromLat: number,
    fromLon: number,
    toLat: number,
    toLon: number
): {
    distanceKm: number;
    estimatedHours: number;
    estimatedDuration: string;
} {
    const distanceKm = getDistanceFromLatLonInKm(fromLat, fromLon, toLat, toLon);
    const flight = estimateFlightTime(distanceKm);

    return {
        distanceKm,
        estimatedHours: flight.hours,
        estimatedDuration: flight.formatted,
    };
}

/**
 * 计算港口间距离和预计航行时间
 */
export function calculatePortDistance(
    fromLat: number,
    fromLon: number,
    toLat: number,
    toLon: number
): {
    distanceKm: number;
    distanceNm: number;
    estimatedDays: number;
    estimatedDuration: string;
} {
    const { km, nm } = calculateDistances(fromLat, fromLon, toLat, toLon);
    const sea = estimateSeaTime(nm);

    return {
        distanceKm: km,
        distanceNm: nm,
        estimatedDays: sea.days,
        estimatedDuration: sea.formatted,
    };
}

// ============================================================
// 时区处理工具 (Timezone Utilities)
// 建议：始终使用UTC时间戳进行差值计算，最后再转回当地时间展示
// ============================================================

/**
 * 计算预计到达时间 (ETA)
 * 使用UTC时间戳进行计算，避免跨越国际日期变更线时的日期陷阱
 * @param departureTime 出发时间 (Date对象或ISO字符串)
 * @param flightHours 预计飞行时间(小时)
 * @returns ETA的Date对象和格式化字符串
 */
export function calculateETA(
    departureTime: Date | string,
    flightHours: number
): { eta: Date; etaISO: string } {
    const departure = typeof departureTime === 'string'
        ? new Date(departureTime)
        : departureTime;

    // 使用UTC毫秒时间戳计算
    const etaTimestamp = departure.getTime() + (flightHours * 60 * 60 * 1000);
    const eta = new Date(etaTimestamp);

    return {
        eta,
        etaISO: eta.toISOString(),
    };
}

/**
 * 格式化时间为指定时区
 * @param date Date对象
 * @param timezone IANA时区标识，如 'Asia/Shanghai'、'America/Los_Angeles'
 * @returns 本地化的时间字符串
 */
export function formatInTimezone(
    date: Date,
    timezone: string
): string {
    try {
        return date.toLocaleString('zh-CN', {
            timeZone: timezone,
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
        });
    } catch {
        // 如果时区无效，返回UTC时间
        return date.toISOString();
    }
}

/**
 * 计算空运航班的完整ETA信息
 * @param departureTime 出发时间
 * @param departureTimezone 出发机场时区
 * @param distanceKm 距离(公里)
 * @param arrivalTimezone 到达机场时区
 * @returns 完整的ETA信息
 */
export function calculateFlightETA(
    departureTime: Date | string,
    departureTimezone: string,
    distanceKm: number,
    arrivalTimezone: string
): {
    eta: Date;
    etaLocal: string;
    etaDepartureTz: string;
    flightHours: number;
    actualDistanceKm: number;
} {
    const flight = estimateFlightTime(distanceKm);
    const { eta } = calculateETA(departureTime, flight.hours);

    return {
        eta,
        etaLocal: formatInTimezone(eta, arrivalTimezone),
        etaDepartureTz: formatInTimezone(eta, departureTimezone),
        flightHours: flight.hours,
        actualDistanceKm: flight.actualDistanceKm,
    };
}
