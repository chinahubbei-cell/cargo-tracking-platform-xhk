package services

import (
	"encoding/json"
	"math"
)

// IsPointInPolygon 使用射线法(Ray Casting)判断点是否在多边形内
// lat, lng: 待检测点的纬度和经度
// polygonJSON: 多边形顶点的JSON数组 "[[lat1,lng1],[lat2,lng2],...]"
func IsPointInPolygon(lat, lng float64, polygonJSON string) bool {
	if polygonJSON == "" {
		return false
	}

	var polygon [][]float64
	if err := json.Unmarshal([]byte(polygonJSON), &polygon); err != nil {
		return false
	}

	return isPointInPolygonArray(lat, lng, polygon)
}

// isPointInPolygonArray 射线法核心算法
func isPointInPolygonArray(lat, lng float64, polygon [][]float64) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}

	inside := false
	j := n - 1

	for i := 0; i < n; i++ {
		// polygon[i] = [lat, lng]
		yi, xi := polygon[i][0], polygon[i][1]
		yj, xj := polygon[j][0], polygon[j][1]

		// 射线法：从点向右发射一条水平射线，计算与多边形边的交点数
		if ((yi > lat) != (yj > lat)) &&
			(lng < (xj-xi)*(lat-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// IsPointInCircle 判断点是否在圆形围栏内
func IsPointInCircle(lat, lng, centerLat, centerLng float64, radiusMeters int) bool {
	distance := HaversineDistance(lat, lng, centerLat, centerLng)
	return distance <= float64(radiusMeters)
}

// CalculatePolygonCenter 计算多边形中心点
func CalculatePolygonCenter(polygonJSON string) (float64, float64) {
	if polygonJSON == "" {
		return 0, 0
	}

	var polygon [][]float64
	if err := json.Unmarshal([]byte(polygonJSON), &polygon); err != nil {
		return 0, 0
	}

	if len(polygon) == 0 {
		return 0, 0
	}

	var sumLat, sumLng float64
	for _, point := range polygon {
		sumLat += point[0]
		sumLng += point[1]
	}

	n := float64(len(polygon))
	return sumLat / n, sumLng / n
}

// CalculatePolygonArea 计算多边形面积(平方米) - 使用测量学公式
func CalculatePolygonArea(polygonJSON string) float64 {
	if polygonJSON == "" {
		return 0
	}

	var polygon [][]float64
	if err := json.Unmarshal([]byte(polygonJSON), &polygon); err != nil {
		return 0
	}

	n := len(polygon)
	if n < 3 {
		return 0
	}

	// 使用Shoelace公式计算面积
	const R = 6371000 // 地球半径(米)
	var area float64

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		lat1 := polygon[i][0] * math.Pi / 180
		lng1 := polygon[i][1] * math.Pi / 180
		lat2 := polygon[j][0] * math.Pi / 180
		lng2 := polygon[j][1] * math.Pi / 180

		area += (lng2 - lng1) * (2 + math.Sin(lat1) + math.Sin(lat2))
	}

	area = math.Abs(area * R * R / 2)
	return area
}
