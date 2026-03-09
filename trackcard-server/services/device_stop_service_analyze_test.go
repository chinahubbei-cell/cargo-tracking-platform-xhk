package services

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"trackcard-server/models"
)

func newDeviceStopAnalyzeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:device_stop_analyze_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.DeviceTrack{}, &models.DeviceStopRecord{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return db
}

func primeStopAddressCache(lat, lng float64) {
	stopAddressCache.Store(fmt.Sprintf("%.4f,%.4f", lat, lng), "测试地址 / Test Address")
}

func TestAnalyzeDeviceTracksAndCreateStops_RecoversActiveFromLowSpeedPoints(t *testing.T) {
	db := newDeviceStopAnalyzeTestDB(t)
	svc := NewDeviceStopService(db)

	deviceID := "dev-1"
	externalID := "ext-1"
	shipmentID := "ship-1"
	lat, lng := 55.127052, 37.552547
	primeStopAddressCache(lat, lng)

	base := time.Now().Add(-40 * time.Minute).Truncate(time.Second)
	tracks := []models.DeviceTrack{
		{DeviceID: deviceID, Latitude: lat, Longitude: lng, Speed: 2, LocateTime: base},
		{DeviceID: deviceID, Latitude: lat, Longitude: lng, Speed: 2, LocateTime: base.Add(10 * time.Minute)},
		{DeviceID: deviceID, Latitude: lat, Longitude: lng, Speed: 2, LocateTime: base.Add(20 * time.Minute)},
	}
	if err := db.Create(&tracks).Error; err != nil {
		t.Fatalf("seed tracks failed: %v", err)
	}

	if err := svc.AnalyzeDeviceTracksAndCreateStops(deviceID, externalID, shipmentID, base.Add(-time.Minute), base.Add(21*time.Minute)); err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	var records []models.DeviceStopRecord
	if err := db.Where("shipment_id = ?", shipmentID).Order("start_time ASC").Find(&records).Error; err != nil {
		t.Fatalf("query records failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 stop record, got %d", len(records))
	}
	if records[0].Status != "active" {
		t.Fatalf("expected active status, got %s", records[0].Status)
	}
	if records[0].EndTime != nil {
		t.Fatalf("expected nil end_time for active record, got %v", *records[0].EndTime)
	}
	if records[0].DurationSeconds <= 0 {
		t.Fatalf("expected positive duration, got %d", records[0].DurationSeconds)
	}
}

func TestAnalyzeDeviceTracksAndCreateStops_KeepsExistingActiveOnLowSpeedTail(t *testing.T) {
	db := newDeviceStopAnalyzeTestDB(t)
	svc := NewDeviceStopService(db)

	deviceID := "dev-2"
	externalID := "ext-2"
	shipmentID := "ship-2"
	lat, lng := 55.127052, 37.552547
	primeStopAddressCache(lat, lng)

	activeStart := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	existing := models.DeviceStopRecord{
		DeviceID:         deviceID,
		DeviceExternalID: externalID,
		ShipmentID:       shipmentID,
		StartTime:        activeStart,
		Latitude:         &lat,
		Longitude:        &lng,
		Address:          "测试地址 / Test Address",
		Status:           "active",
		DurationSeconds:  0,
		DurationText:     "0秒",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed active record failed: %v", err)
	}

	trackTime := time.Now().Add(-5 * time.Minute).Truncate(time.Second)
	track := models.DeviceTrack{
		DeviceID:   deviceID,
		Latitude:   lat,
		Longitude:  lng,
		Speed:      2,
		LocateTime: trackTime,
	}
	if err := db.Create(&track).Error; err != nil {
		t.Fatalf("seed low-speed track failed: %v", err)
	}

	if err := svc.AnalyzeDeviceTracksAndCreateStops(deviceID, externalID, shipmentID, trackTime.Add(-30*time.Minute), trackTime.Add(time.Minute)); err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	var after models.DeviceStopRecord
	if err := db.First(&after, "id = ?", existing.ID).Error; err != nil {
		t.Fatalf("query active record failed: %v", err)
	}

	if after.Status != "active" {
		t.Fatalf("expected active status, got %s", after.Status)
	}
	if after.EndTime != nil {
		t.Fatalf("expected nil end_time, got %v", *after.EndTime)
	}
}

func TestAnalyzeDeviceTracksAndCreateStops_SkipsSingleLowSpeedPointSegments(t *testing.T) {
	db := newDeviceStopAnalyzeTestDB(t)
	svc := NewDeviceStopService(db)

	deviceID := "dev-3"
	externalID := "ext-3"
	shipmentID := "ship-3"

	aLat, aLng := 39.9000, 116.4000
	bLat, bLng := 39.9030, 116.4000 // 与 A 相距约 333 米
	primeStopAddressCache(aLat, aLng)
	primeStopAddressCache(bLat, bLng)

	base := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	tracks := []models.DeviceTrack{
		{DeviceID: deviceID, Latitude: aLat, Longitude: aLng, Speed: 2, LocateTime: base},
		{DeviceID: deviceID, Latitude: bLat, Longitude: bLng, Speed: 2, LocateTime: base.Add(10 * time.Minute)},
		{DeviceID: deviceID, Latitude: bLat, Longitude: bLng, Speed: 35, LocateTime: base.Add(20 * time.Minute)},
	}
	if err := db.Create(&tracks).Error; err != nil {
		t.Fatalf("seed tracks failed: %v", err)
	}

	if err := svc.AnalyzeDeviceTracksAndCreateStops(deviceID, externalID, shipmentID, base.Add(-time.Minute), base.Add(21*time.Minute)); err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.DeviceStopRecord{}).Where("shipment_id = ?", shipmentID).Count(&count).Error; err != nil {
		t.Fatalf("count records failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 stop records, got %d", count)
	}
}
