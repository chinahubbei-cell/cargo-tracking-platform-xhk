package models

// 标准里程碑序列 (国际海运)
// 用于 Phase 2 船司追踪的里程碑排序
var SeaMilestoneSequence = map[string]int{
	"BOOKING_CONFIRMED": 10,
	"GATE_OUT":          20,
	"LOADED":            30,
	"VESSEL_DEPARTURE":  40,
	"TRANSSHIPMENT":     50,
	"VESSEL_ARRIVAL":    60,
	"DISCHARGE":         70,
	"CUSTOMS_HOLD":      75,
	"CUSTOMS_RELEASE":   80,
	"GATE_IN":           85,
	"DELIVERY":          90,
	"COMPLETED":         100,
}

// MilestoneStatus 里程碑状态 (Phase 2)
const (
	MilestoneStatusPlanned = "planned"
	MilestoneStatusActual  = "actual"
	MilestoneStatusSkipped = "skipped"
)

// MilestoneSource 里程碑数据源 (Phase 2)
const (
	MilestoneSourceIOT      = "iot"
	MilestoneSourceCarrier  = "carrier"
	MilestoneSourceManual   = "manual"
	MilestoneSourceGeofence = "geofence"
)
