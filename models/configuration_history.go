package models

import (
	"time"
)

// ConfigurationHistory corresponds to the `configuration_history` table.
type ConfigurationHistory struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigurationID uint      `gorm:"not null;index:idx_hist_config_id" json:"configuration_id"` // Foreign Key referencing Configurations
	Content         string    `gorm:"type:text;not null" json:"content"`
	Format          string    `gorm:"type:varchar(50);not null" json:"format"`               // Historical format
	Version         uint      `gorm:"not null;index:idx_hist_config_version" json:"version"` // The version number this history entry represents
	SavedBy         string    `json:"saved_by,omitempty"`                                    // Identifier of the user/system saving it
	SavedAt         time.Time `json:"saved_at"`

	// Foreign key constraint (optional, GORM can work without it but good for DB integrity)
	// Configuration   Configuration `gorm:"foreignKey:ConfigurationID"`
}

// TableName specifies the table name for GORM.
func (ConfigurationHistory) TableName() string {
	return "configuration_history"
}
