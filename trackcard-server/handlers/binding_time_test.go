package handlers

import (
	"testing"
	"time"

	"trackcard-server/models"
)

func TestResolveBindingStartTimePrefersValidBindingCreatedAt(t *testing.T) {
	shipmentCreatedAt := time.Date(2026, 2, 25, 15, 59, 13, 0, time.FixedZone("CST", 8*3600))
	bindingCreatedAt := shipmentCreatedAt.Add(2 * time.Second)

	shipment := &models.Shipment{
		CreatedAt: shipmentCreatedAt,
	}
	binding := models.ShipmentDeviceBinding{
		BoundAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		CreatedAt: bindingCreatedAt,
	}

	got := resolveBindingStartTime(shipment, binding)
	if !got.Equal(bindingCreatedAt) {
		t.Fatalf("expected %v, got %v", bindingCreatedAt, got)
	}
}

func TestResolveBindingStartTimeClampsToShipmentCreatedAt(t *testing.T) {
	shipmentCreatedAt := time.Date(2026, 2, 25, 15, 59, 13, 0, time.FixedZone("CST", 8*3600))
	deviceBoundAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))

	shipment := &models.Shipment{
		CreatedAt:     shipmentCreatedAt,
		DeviceBoundAt: &deviceBoundAt,
	}
	binding := models.ShipmentDeviceBinding{}

	got := resolveBindingStartTime(shipment, binding)
	if !got.Equal(shipmentCreatedAt) {
		t.Fatalf("expected %v, got %v", shipmentCreatedAt, got)
	}
}

func TestResolveBindingStartTimeKeepsValidBoundAt(t *testing.T) {
	shipmentCreatedAt := time.Date(2026, 2, 25, 15, 59, 13, 0, time.FixedZone("CST", 8*3600))
	boundAt := shipmentCreatedAt.Add(10 * time.Minute)

	shipment := &models.Shipment{
		CreatedAt: shipmentCreatedAt,
	}
	binding := models.ShipmentDeviceBinding{
		BoundAt: boundAt,
	}

	got := resolveBindingStartTime(shipment, binding)
	if !got.Equal(boundAt) {
		t.Fatalf("expected %v, got %v", boundAt, got)
	}
}
