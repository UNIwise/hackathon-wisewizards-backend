package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var Migration00001init = &gormigrate.Migration{
	ID: "00001_init",
	Migrate: func(tx *gorm.DB) error {
		type Flow struct {
			gorm.Model
			Title string `gorm:"index;not null"`
		}

		return tx.AutoMigrate(&Flow{})
	},
	Rollback: func(tx *gorm.DB) error {
		return tx.Migrator().DropTable("flows")
	},
}
