package handlers

import (
	"time"

	"trackcard-server/models"
)

// resolveBindingStartTime normalizes binding start time for track queries.
// It guards against dirty historical data where bound_at may be earlier than
// shipment creation or binding row creation.
func resolveBindingStartTime(shipment *models.Shipment, binding models.ShipmentDeviceBinding) time.Time {
	start := binding.BoundAt
	if start.IsZero() {
		if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() {
			start = *shipment.DeviceBoundAt
		} else if shipment.LeftOriginAt != nil && !shipment.LeftOriginAt.IsZero() {
			start = *shipment.LeftOriginAt
		} else {
			start = shipment.CreatedAt
		}
	}

	if !shipment.CreatedAt.IsZero() && start.Before(shipment.CreatedAt) {
		start = shipment.CreatedAt
	}
	if !binding.CreatedAt.IsZero() && start.Before(binding.CreatedAt) {
		start = binding.CreatedAt
	}

	return start
}
