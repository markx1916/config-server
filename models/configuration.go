package models

import (
	"time"
)

// Configuration corresponds to the `configurations` table.
type Configuration struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NacosInstanceID uint      `gorm:"not null;index:idx_config_nacos_instance" json:"nacos_instance_id"`
	DataID          string    `gorm:"type:varchar(255);not null;index:idx_config_data_id" json:"data_id"`
	GroupName       string    `gorm:"type:varchar(255);not null;index:idx_config_group_name" json:"group_name"`
	Content         string    `gorm:"type:text;not null" json:"content"`
	Format          string    `gorm:"type:varchar(50);not null;default:'text'" json:"format"` // e.g., yaml, json, properties, text
	Description     string    `gorm:"type:text" json:"description,omitempty"`
	Version         uint      `gorm:"not null;default:1" json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Foreign key constraint (optional, GORM can work without it but good for DB integrity)
	// NacosInstance NacosInstance `gorm:"foreignKey:NacosInstanceID"`
}

// TableName specifies the table name for GORM.
func (Configuration) TableName() string {
	return "configurations"
}

// CreateConfigurationRequest defines the structure for creating a new configuration.
type CreateConfigurationRequest struct {
	NacosInstanceID uint   `json:"nacos_instance_id" binding:"required"`
	DataID          string `json:"data_id" binding:"required"`
	GroupName       string `json:"group_name" binding:"required"`
	Content         string `json:"content" binding:"required"`
	Format          string `json:"format" binding:"required,oneof=text json yaml properties xml html"`
	Description     string `json:"description,omitempty"`
}

// UpdateConfigurationRequest defines the structure for updating an existing configuration.
type UpdateConfigurationRequest struct {
	Content     *string `json:"content,omitempty"`
	Format      *string `json:"format,omitempty"` // Consider if format changes should be allowed or versioned differently
	Description *string `json:"description,omitempty"`
}

// ConfigurationResponse is used to structure the response for a single configuration.
// It can be the same as Configuration or a subset if needed.
type ConfigurationResponse struct {
	ID              uint      `json:"id"`
	NacosInstanceID uint      `json:"nacos_instance_id"`
	DataID          string    `json:"data_id"`
	GroupName       string    `json:"group_name"`
	Content         string    `json:"content"`
	Format          string    `json:"format"`
	Description     string    `json:"description,omitempty"`
	Version         uint      `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DiffResponse defines the structure for the diff endpoint.
type DiffResponse struct {
	CurrentVersionContent  string `json:"current_version_content"`
	ComparedVersion        uint   `json:"compared_version"`
	ComparedVersionContent string `json:"compared_version_content"`
	DiffOutput             string `json:"diff_output"` // For now, a simple textual representation or combined content
}
