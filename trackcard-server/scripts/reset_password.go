package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model simplified for password reset
type User struct {
	ID       string
	Email    string
	Password string
}

func main() {
	// 默认认为在 trackcard-server 根目录下运行
	dbPath := "trackcard.db"

	// 如果由于某种原因是在 scripts 目录下运行
	// dbPath = "../trackcard.db"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	email := "admin@trackcard.com"
	newPassword := "admin123"

	// Check if user exists
	var user User
	result := db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("用户 %s 不存在，正在创建...", email)
			// Create the user if not exists
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			user = User{
				ID:       "user-admin",
				Email:    email,
				Password: string(hashedPassword),
			}
			if err := db.Create(&user).Error; err != nil {
				log.Fatalf("创建用户失败: %v", err)
			}
			log.Println("用户创建成功！")
			return
		}
		log.Fatalf("查询用户失败: %v", result.Error)
	}

	// Update password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("密码加密失败: %v", err)
	}

	if err := db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		log.Fatalf("重置密码失败: %v", err)
	}

	fmt.Printf("✅ 用户 %s 的密码已重置为: %s\n", email, newPassword)
}
