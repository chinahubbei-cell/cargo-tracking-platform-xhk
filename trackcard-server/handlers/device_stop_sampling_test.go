package handlers

import (
	"testing"
	"time"

	"trackcard-server/models"
)

func TestSelectTransitTrackSamples_DistributesAcrossFullTimeline(t *testing.T) {
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	tracks := make([]models.DeviceTrack, 0, 600)
	for i := 0; i < 600; i++ {
		tracks = append(tracks, models.DeviceTrack{
			DeviceID:   "dev-1",
			Latitude:   float64(i), // 每点纬度差 1 度，触发距离阈值
			Longitude:  110.0,
			LocateTime: base.Add(time.Duration(i) * time.Minute),
		})
	}

	samples := selectTransitTrackSamples(tracks, 180)
	if len(samples) != 180 {
		t.Fatalf("unexpected sample count: got=%d want=180", len(samples))
	}

	if !samples[0].LocateTime.Equal(tracks[0].LocateTime) {
		t.Fatalf("first sample should keep first track")
	}
	if !samples[len(samples)-1].LocateTime.Equal(tracks[len(tracks)-1].LocateTime) {
		t.Fatalf("last sample should keep last track")
	}

	middleSample := samples[len(samples)/2]
	middleThreshold := base.Add(250 * time.Minute)
	if middleSample.LocateTime.Before(middleThreshold) {
		t.Fatalf("samples are biased to early timeline: middle=%s threshold=%s", middleSample.LocateTime, middleThreshold)
	}
}

func TestDownsampleTransitTracks_PreservesBoundary(t *testing.T) {
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	tracks := make([]models.DeviceTrack, 0, 25)
	for i := 0; i < 25; i++ {
		tracks = append(tracks, models.DeviceTrack{
			DeviceID:   "dev-2",
			Latitude:   30.0 + float64(i)*0.01,
			Longitude:  110.0,
			LocateTime: base.Add(time.Duration(i) * time.Minute),
		})
	}

	samples := downsampleTransitTracks(tracks, 8)
	if len(samples) != 8 {
		t.Fatalf("unexpected sample count: got=%d want=8", len(samples))
	}
	if !samples[0].LocateTime.Equal(tracks[0].LocateTime) {
		t.Fatalf("first sample should keep first track")
	}
	if !samples[len(samples)-1].LocateTime.Equal(tracks[len(tracks)-1].LocateTime) {
		t.Fatalf("last sample should keep last track")
	}
}
