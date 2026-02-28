package services

import (
	"fmt"
	"math/rand"
	"time"
	"trackcard-server/models"
)

// MockCarrierProvider 模拟船司API提供商 (用于开发和测试)
type MockCarrierProvider struct {
	enabled bool
}

// NewMockCarrierProvider 创建模拟提供商
func NewMockCarrierProvider(enabled bool) *MockCarrierProvider {
	return &MockCarrierProvider{enabled: enabled}
}

func (p *MockCarrierProvider) Name() string {
	return "mock"
}

func (p *MockCarrierProvider) IsConfigured() bool {
	return p.enabled
}

// TrackByBL 模拟根据提单号查询事件
func (p *MockCarrierProvider) TrackByBL(billOfLading string) ([]CarrierEvent, error) {
	if !p.enabled {
		return nil, fmt.Errorf("mock provider disabled")
	}

	// 生成模拟事件 (基于提单号生成确定性数据)
	seed := int64(0)
	for _, c := range billOfLading {
		seed += int64(c)
	}
	r := rand.New(rand.NewSource(seed))

	// 模拟船务信息
	carriers := []string{"COSCO", "MAERSK", "MSC", "CMA-CGM", "EVERGREEN"}
	vessels := []string{"EVER GIVEN", "MSC GULSUN", "CMA CGM ANTOINE", "COSCO SHIPPING UNIVERSE"}
	ports := []struct {
		name   string
		locode string
		lat    float64
		lng    float64
	}{
		{"Shanghai, China", "CNSHA", 31.2304, 121.4737},
		{"Los Angeles, USA", "USLAX", 33.7367, -118.2793},
		{"Rotterdam, Netherlands", "NLRTM", 51.9225, 4.4792},
		{"Singapore", "SGSIN", 1.2644, 103.8200},
		{"Hamburg, Germany", "DEHAM", 53.5511, 9.9937},
	}

	carrier := carriers[r.Intn(len(carriers))]
	vessel := vessels[r.Intn(len(vessels))]
	voyage := fmt.Sprintf("%dE%02d", time.Now().Year()%100, r.Intn(50)+1)
	originPort := ports[r.Intn(len(ports))]
	destPort := ports[r.Intn(len(ports))]
	for destPort.locode == originPort.locode {
		destPort = ports[r.Intn(len(ports))]
	}

	// 生成事件时间线
	now := time.Now()
	baseTime := now.AddDate(0, 0, -r.Intn(30)) // 过去30天内开始

	events := []CarrierEvent{
		{
			EventCode:  models.EventGateOut,
			EventName:  models.EventCodeToName[models.EventGateOut],
			Location:   originPort.name,
			LoCode:     originPort.locode,
			Latitude:   &originPort.lat,
			Longitude:  &originPort.lng,
			VesselName: vessel,
			VoyageNo:   voyage,
			Carrier:    carrier,
			EventTime:  baseTime,
			IsActual:   true,
		},
		{
			EventCode:  models.EventLoadedOnVessel,
			EventName:  models.EventCodeToName[models.EventLoadedOnVessel],
			Location:   originPort.name,
			LoCode:     originPort.locode,
			VesselName: vessel,
			VoyageNo:   voyage,
			Carrier:    carrier,
			EventTime:  baseTime.Add(24 * time.Hour),
			IsActual:   true,
		},
		{
			EventCode:  models.EventVesselDeparture,
			EventName:  models.EventCodeToName[models.EventVesselDeparture],
			Location:   originPort.name,
			LoCode:     originPort.locode,
			Latitude:   &originPort.lat,
			Longitude:  &originPort.lng,
			VesselName: vessel,
			VoyageNo:   voyage,
			Carrier:    carrier,
			EventTime:  baseTime.Add(48 * time.Hour),
			IsActual:   true,
		},
	}

	// 如果已经过去足够时间，添加到港事件
	transitDays := 14 + r.Intn(21) // 14-35天
	arrivalTime := baseTime.Add(time.Duration(transitDays*24) * time.Hour)

	if now.After(arrivalTime) {
		events = append(events, CarrierEvent{
			EventCode:  models.EventVesselArrival,
			EventName:  models.EventCodeToName[models.EventVesselArrival],
			Location:   destPort.name,
			LoCode:     destPort.locode,
			Latitude:   &destPort.lat,
			Longitude:  &destPort.lng,
			VesselName: vessel,
			VoyageNo:   voyage,
			Carrier:    carrier,
			EventTime:  arrivalTime,
			IsActual:   true,
		})
	} else {
		// 添加预计到达 (ETA)
		eta := arrivalTime
		events = append(events, CarrierEvent{
			EventCode:  models.EventVesselArrival,
			EventName:  models.EventCodeToName[models.EventVesselArrival] + " (预计)",
			Location:   destPort.name,
			LoCode:     destPort.locode,
			Latitude:   &destPort.lat,
			Longitude:  &destPort.lng,
			VesselName: vessel,
			VoyageNo:   voyage,
			Carrier:    carrier,
			EventTime:  arrivalTime,
			ETAUpdate:  &eta,
			IsActual:   false,
		})
	}

	return events, nil
}
