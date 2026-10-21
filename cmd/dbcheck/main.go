package main

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID          uint   `gorm:"primaryKey"`
	Username    string
	Email       string
	PasswordHash string `gorm:"column:password_hash"`
	IsSiteAdmin bool   `gorm:"column:is_site_admin"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("data/app.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("open db error:", err)
		os.Exit(1)
	}
	var users []User
	db.Find(&users)
	for _, u := range users {
		fmt.Printf("ID=%d  user=%-15s  admin=%v  hash_prefix=%s...\n",
			u.ID, u.Username, u.IsSiteAdmin, u.PasswordHash[:min(20, len(u.PasswordHash))])
	}
}

func min(a, b int) int { if a < b { return a }; return b }
