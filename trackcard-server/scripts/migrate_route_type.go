package main

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 判断是否为国内运输
func isDomesticRoute(origin, destination, originAddr, destAddr string) bool {
	domesticKeywords := []string{"中国", "CN", "China", "china", "Mainland China"}

	// 优先使用详细地址
	checkOrigin := originAddr
	if checkOrigin == "" {
		checkOrigin = origin
	}
	checkDest := destAddr
	if checkDest == "" {
		checkDest = destination
	}

	isOriginDomestic := false
	for _, kw := range domesticKeywords {
		if strings.Contains(strings.ToLower(checkOrigin), strings.ToLower(kw)) {
			isOriginDomestic = true
			break
		}
	}

	isDestDomestic := false
	for _, kw := range domesticKeywords {
		if strings.Contains(strings.ToLower(checkDest), strings.ToLower(kw)) {
			isDestDomestic = true
			break
		}
	}

	return isOriginDomestic && isDestDomestic
}

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 查询所有运单
	var shipments []struct {
		ID            string
		Origin        string
		Destination   string
		OriginAddress string `gorm:"column:origin_address"`
		DestAddress   string `gorm:"column:dest_address"`
		RouteType     string `gorm:"column:route_type"`
	}

	if err := db.Table("shipments").Find(&shipments).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("找到 %d 个运单需要处理\n", len(shipments))

	domesticCount := 0
	crossBorderCount := 0

	// 更新每个运单的route_type
	for _, s := range shipments {
		routeType := "cross_border"
		if isDomesticRoute(s.Origin, s.Destination, s.OriginAddress, s.DestAddress) {
			routeType = "domestic"
			domesticCount++
		} else {
			crossBorderCount++
		}

		if err := db.Exec("UPDATE shipments SET route_type = ? WHERE id = ?", routeType, s.ID).Error; err != nil {
			log.Printf("更新运单 %s 失败: %v", s.ID, err)
		} else {
			fmt.Printf("运单 %s: %s -> %s (类型: %s)\n", s.ID, s.Origin, s.Destination, routeType)
		}
	}

	fmt.Printf("\n迁移完成！\n")
	fmt.Printf("境内运输: %d\n", domesticCount)
	fmt.Printf("跨境运输: %d\n", crossBorderCount)
}
