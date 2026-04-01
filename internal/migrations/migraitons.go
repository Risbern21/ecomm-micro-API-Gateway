package migrations

import (
	"log"

	"github.com/risbern21/api_gateway/internal/database"
	"github.com/risbern21/api_gateway/model"
)

func AutoMigrate() {
	if err := database.Client().AutoMigrate(&model.User{}, &model.Session{}); err != nil {
		log.Fatalf("unable to migrate %v", err)
	}
}
