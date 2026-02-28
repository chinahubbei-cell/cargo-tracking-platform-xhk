package main

import (
	"fmt"
	"trackcard-server/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	password := "admin123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	var user models.User
	if err := db.Where("email = ?", "admin@trackcard.com").First(&user).Error; err != nil {
		fmt.Println("User not found")
		return
	}

	user.Password = string(hashedPassword)
	db.Save(&user)
	fmt.Printf("Password for %s reset to %s (hash: %s)\n", user.Email, password, user.Password)
}
