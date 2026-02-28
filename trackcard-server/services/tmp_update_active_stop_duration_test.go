package services

import (
    "os"
    "testing"

    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "trackcard-server/models"
)

func TestTmpUpdateActiveStopDuration(t *testing.T) {
    dbPath := "../trackcard.db"
    if _, err := os.Stat(dbPath); err != nil {
        t.Fatalf("db not found: %v", err)
    }

    db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
    if err != nil {
        t.Fatalf("open db: %v", err)
    }

    var before models.DeviceStopRecord
    if err := db.Where("id = ?", "DSR-af4309d1").First(&before).Error; err != nil {
        t.Fatalf("load before: %v", err)
    }

    svc := NewDeviceStopService(db)
    if err := svc.UpdateActiveStopDurations(); err != nil {
        t.Fatalf("UpdateActiveStopDurations failed: %v", err)
    }

    var after models.DeviceStopRecord
    if err := db.Where("id = ?", "DSR-af4309d1").First(&after).Error; err != nil {
        t.Fatalf("load after: %v", err)
    }

    t.Logf("before duration=%d updated_at=%s", before.DurationSeconds, before.UpdatedAt)
    t.Logf("after  duration=%d updated_at=%s", after.DurationSeconds, after.UpdatedAt)

    if after.DurationSeconds < before.DurationSeconds {
        t.Fatalf("duration decreased: before=%d after=%d", before.DurationSeconds, after.DurationSeconds)
    }
    if after.DurationSeconds == before.DurationSeconds {
        t.Fatalf("duration not changed: before=%d after=%d", before.DurationSeconds, after.DurationSeconds)
    }
}
