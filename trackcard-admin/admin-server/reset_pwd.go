package main

import (
"fmt"
"trackcard-admin/models"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("../../trackcard-server/trackcard.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}

	var user models.AdminUser
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		fmt.Println("Admin user not found:", err)
		return
	}

	user.SetPassword("admin123")
	if err := db.Save(&user).Error; err != nil {
		fmt.Println("Failed to save user:", err)
		return
	}

	fmt.Println("Admin password successfully reset to 'admin123'")
}
