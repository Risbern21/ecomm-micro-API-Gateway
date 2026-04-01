package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func Client() *gorm.DB {
	return db
}

func Setup() {
	var err error

	dsn := os.Getenv("DSN")

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Unable to connect to database : %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("Unable to ping the database %v", err)
	}

	log.Println("Successfully connected to the database")
}
