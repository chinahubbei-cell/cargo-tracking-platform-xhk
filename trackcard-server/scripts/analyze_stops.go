package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DeviceTrack struct {
	ID         int
	DeviceID   string
	LocateTime time.Time
	Speed      float64
	Latitude   float64
	Longitude  float64
}

type StopSegment struct {
	DeviceID    string
	StartTime   time.Time
	EndTime     *time.Time
	Latitude    float64
	Longitude   float64
	TrackCount  int
}

func main() {
	// 打开数据库
	db, err := sql.Open("sqlite3", "trackcard.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 要分析的设备ID
	deviceIDs := []string{"GC-83fd51e3", "GC-1f715b21"}

	// 查询运单ID
	shipmentIDs := map[string]string{
		"GC-83fd51e3": "260128000003",
		"GC-1f715b21": "260128000004",
	}

	// 查询外部设备ID
	externalDeviceIDs := map[string]string{
		"GC-83fd51e3":  "868120343595788",
		"GC-1f715b21": "868120343599970",
	}

	for _, deviceID := range deviceIDs {
		fmt.Printf("\n=== 分析设备 %s ===\n", deviceID)

		// 查询轨迹数据，按时间排序
		rows, err := db.Query(`
			SELECT id, device_id, locate_time, speed, latitude, longitude
			FROM device_tracks
			WHERE device_id = ?
			ORDER BY locate_time ASC
		`, deviceID)
		if err != nil {
			log.Printf("查询轨迹数据失败: %v", err)
			continue
		}

		var tracks []DeviceTrack
		for rows.Next() {
			var t DeviceTrack
			err := rows.Scan(&t.ID, &t.DeviceID, &t.LocateTime, &t.Speed, &t.Latitude, &t.Longitude)
			if err != nil {
				log.Printf("扫描轨迹数据失败: %v", err)
				continue
			}
			tracks = append(tracks, t)
		}
		rows.Close()

		if len(tracks) == 0 {
			fmt.Println("没有找到轨迹数据")
			continue
		}

		fmt.Printf("共 %d 条轨迹记录\n", len(tracks))
		fmt.Printf("时间范围: %s -> %s\n", tracks[0].LocateTime.Format("2006-01-02 15:04:05"),
			tracks[len(tracks)-1].LocateTime.Format("2006-01-02 15:04:05"))

		// 分析停留时段（速度为0且持续超过5分钟）
		var stops []StopSegment
		var currentStop *StopSegment
		minStopDuration := 5 * time.Minute
		locationChangeThreshold := 0.1 // 降低阈值到100米，更容易识别新停留

		for _, track := range tracks {
			// 判断是否为停止状态（速度=0）
			if track.Speed == 0 {
				if currentStop == nil {
					// 开始新的停留
					currentStop = &StopSegment{
						DeviceID:   track.DeviceID,
						StartTime:  track.LocateTime,
						Latitude:   track.Latitude,
						Longitude:  track.Longitude,
						TrackCount: 1,
					}
				} else {
					// 判断位置变化（超过阈值则认为是新的停留）
					dist := haversine(currentStop.Latitude, currentStop.Longitude,
						track.Latitude, track.Longitude)
					if dist > locationChangeThreshold {
						// 位置变化较大，结束当前停留并开始新的停留
						if currentStop.TrackCount > 0 {
							duration := track.LocateTime.Sub(currentStop.StartTime)
							if duration >= minStopDuration {
								endTime := track.LocateTime
								currentStop.EndTime = &endTime
								stops = append(stops, *currentStop)
							}
						}
						currentStop = &StopSegment{
							DeviceID:   track.DeviceID,
							StartTime:  track.LocateTime,
							Latitude:   track.Latitude,
							Longitude:  track.Longitude,
							TrackCount: 1,
						}
					} else {
						// 继续当前停留
						currentStop.TrackCount++
						// 更新位置为平均位置
						currentStop.Latitude = (currentStop.Latitude + track.Latitude) / 2
						currentStop.Longitude = (currentStop.Longitude + track.Longitude) / 2
					}
				}
			} else {
				// 速度不为0，结束当前停留
				if currentStop != nil && currentStop.TrackCount > 0 {
					duration := track.LocateTime.Sub(currentStop.StartTime)
					if duration >= minStopDuration {
						endTime := track.LocateTime
						currentStop.EndTime = &endTime
						stops = append(stops, *currentStop)
					}
					currentStop = nil
				}
			}
		}

		// 处理最后一个停留（可能仍在进行中）
		if currentStop != nil && currentStop.TrackCount > 0 {
			duration := time.Since(currentStop.StartTime)
			if duration >= minStopDuration {
				// 没有结束时间，说明仍在停留中
				stops = append(stops, *currentStop)
			}
		}

		fmt.Printf("\n发现 %d 个停留时段:\n", len(stops))

		// 删除旧的模拟数据
		deleteResult, err := db.Exec(`
			DELETE FROM device_stop_records
			WHERE device_id = ? AND shipment_id = ?
		`, externalDeviceIDs[deviceID], shipmentIDs[deviceID])
		if err != nil {
			log.Printf("删除旧数据失败: %v", err)
		} else {
			rowsAffected, _ := deleteResult.RowsAffected()
			fmt.Printf("已删除 %d 条旧记录\n", rowsAffected)
		}

		// 插入新的停留记录
		insertStmt, err := db.Prepare(`
			INSERT INTO device_stop_records
			(id, device_external_id, device_id, shipment_id, start_time, end_time,
			 duration_seconds, duration_text, latitude, longitude, address, status,
			 alert_sent, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			log.Printf("准备插入语句失败: %v", err)
			continue
		}
		defer insertStmt.Close()

		now := time.Now()
		insertedCount := 0

		for i, stop := range stops {
			// 计算停留时长
			var durationSeconds int
			var durationText string
			var endTime *time.Time

			if stop.EndTime != nil {
				duration := stop.EndTime.Sub(stop.StartTime)
				durationSeconds = int(duration.Seconds())
				durationText = formatDuration(durationSeconds)
				endTime = stop.EndTime
			} else {
				duration := now.Sub(stop.StartTime)
				durationSeconds = int(duration.Seconds())
				durationText = formatDuration(durationSeconds)
				endTime = nil
			}

			// 跳过时长小于5分钟的停留
			if durationSeconds < 300 {
				continue
			}

			// 生成唯一ID
			recordID := fmt.Sprintf("%s-%d", deviceID, i)

			// 状态：如果有结束时间则为completed，否则为active
			status := "completed"
			if endTime == nil {
				status = "active"
			}

			// 插入记录
			_, err = insertStmt.Exec(
				recordID,
				externalDeviceIDs[deviceID],
				deviceID,
				shipmentIDs[deviceID],
				stop.StartTime,
				endTime,
				durationSeconds,
				durationText,
				stop.Latitude,
				stop.Longitude,
				"", // 地址暂时为空
				status,
				false,
				now,
				now,
			)

			if err != nil {
				log.Printf("插入记录失败: %v", err)
			} else {
				insertedCount++
				endTimeStr := "进行中"
				if endTime != nil {
					endTimeStr = endTime.Format("2006-01-02 15:04")
				}
				fmt.Printf("  %d. %s -> %s (%s)\n", insertedCount,
					stop.StartTime.Format("2006-01-02 15:04"),
					endTimeStr, durationText)
			}
		}

		fmt.Printf("成功插入 %d 条停留记录\n", insertedCount)
	}

	fmt.Println("\n=== 分析完成 ===")
}

// 计算两点间距离（公里）
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // 地球半径，单位：公里

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Asin(math.Sqrt(a))

	return earthRadius * c
}

// 格式化时长
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%d小时%d分", hours, minutes)
		}
		return fmt.Sprintf("%d小时", hours)
	}

	return fmt.Sprintf("%d分钟", minutes)
}
