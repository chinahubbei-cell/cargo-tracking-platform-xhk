package main

import (
"fmt"
"trackcard-server/models"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}
	var orders []models.Order
	if err := db.Where("org_id = ?", "org-hq").Where("order_type = ?", "purchase").Preload("Items").Find(&orders).Error; err != nil {
		fmt.Println("Query error:", err)
		return
	}
	fmt.Printf("Found %d orders\n", len(orders))
}
