package handlers

import (
	"fmt"
	"testing"
	"time"

	"trackcard-admin/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDeviceSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:device_sync_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err := db.AutoMigrate(&models.HardwareDevice{}, &models.Organization{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			type TEXT,
			name TEXT,
			external_device_id TEXT,
			org_id TEXT,
			created_at DATETIME,
			last_update DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create devices table failed: %v", err)
	}

	return db
}

func TestSyncHardwareDevicesFromBusiness_RepairsOrgOwnership(t *testing.T) {
	db := newDeviceSyncTestDB(t)
	handler := NewDeviceHandler(db)

	if err := db.Create(&models.Organization{ID: "org-hq", Name: "快货运国际集团总部", Level: 1}).Error; err != nil {
		t.Fatalf("create org-hq failed: %v", err)
	}
	if err := db.Create(&models.Organization{ID: "org-east", Name: "华东分公司", ParentID: "org-hq", Level: 2}).Error; err != nil {
		t.Fatalf("create org-east failed: %v", err)
	}

	now := time.Now()
	if err := db.Exec(`
		INSERT INTO devices (id, type, name, external_device_id, org_id, created_at, last_update)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "GC-b564f34a", "container", "设备-86812008209009", "86812008209009", "org-east", now, now).Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	if err := db.Create(&models.HardwareDevice{
		ID:         "GC-b564f34a",
		DeviceType: "container",
		IMEI:       "86812008209009",
		SN:         "GC-b564f34a",
		Status:     "allocated",
		OrgID:      "org-hq",
		OrgName:    "快货运国际集团总部",
	}).Error; err != nil {
		t.Fatalf("insert hardware device failed: %v", err)
	}

	handler.syncHardwareDevicesFromBusiness()

	var got models.HardwareDevice
	if err := db.First(&got, "id = ?", "GC-b564f34a").Error; err != nil {
		t.Fatalf("query hardware device failed: %v", err)
	}

	if got.OrgID != "org-east" {
		t.Fatalf("org_id not repaired, got=%s want=org-east", got.OrgID)
	}
	if got.OrgName != "华东分公司" {
		t.Fatalf("org_name not repaired, got=%s want=华东分公司", got.OrgName)
	}
}

func TestSyncHardwareDevicesFromBusiness_DedupByLatestBusinessRecord(t *testing.T) {
	db := newDeviceSyncTestDB(t)
	handler := NewDeviceHandler(db)

	if err := db.Create(&models.Organization{ID: "org-hq", Name: "快货运国际集团总部", Level: 1}).Error; err != nil {
		t.Fatalf("create org-hq failed: %v", err)
	}
	if err := db.Create(&models.Organization{ID: "org-east", Name: "华东分公司", ParentID: "org-hq", Level: 2}).Error; err != nil {
		t.Fatalf("create org-east failed: %v", err)
	}

	oldAt := time.Now().Add(-2 * time.Hour)
	newAt := time.Now().Add(-10 * time.Minute)

	// 同一 IMEI 存在两条业务记录：旧记录在总部，新记录在华东
	if err := db.Exec(`
		INSERT INTO devices (id, type, name, external_device_id, org_id, created_at, last_update)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "GC-old", "container", "旧设备映射", "86812008203456", "org-hq", oldAt, oldAt).Error; err != nil {
		t.Fatalf("insert old business device failed: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO devices (id, type, name, external_device_id, org_id, created_at, last_update)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "GC-new", "container", "新设备映射", "86812008203456", "org-east", newAt, newAt).Error; err != nil {
		t.Fatalf("insert new business device failed: %v", err)
	}

	if err := db.Create(&models.HardwareDevice{
		ID:         "GC-hw",
		DeviceType: "container",
		IMEI:       "86812008203456",
		SN:         "GC-hw",
		Status:     "allocated",
		OrgID:      "org-hq",
		OrgName:    "快货运国际集团总部",
	}).Error; err != nil {
		t.Fatalf("insert hardware device failed: %v", err)
	}

	handler.syncHardwareDevicesFromBusiness()

	var got models.HardwareDevice
	if err := db.First(&got, "id = ?", "GC-hw").Error; err != nil {
		t.Fatalf("query hardware device failed: %v", err)
	}

	if got.OrgID != "org-east" {
		t.Fatalf("latest business record should win, got org_id=%s want=org-east", got.OrgID)
	}
	if got.OrgName != "华东分公司" {
		t.Fatalf("latest business org name should win, got org_name=%s want=华东分公司", got.OrgName)
	}
}
