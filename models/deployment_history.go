package models

import (
	"time"
)

// DeploymentHistory corresponds to the `deployment_history` table.
type DeploymentHistory struct {
	ID                   uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigurationID      uint      `gorm:"not null;index:idx_deploy_hist_config_id" json:"configuration_id"`
	ConfigurationVersion uint      `gorm:"not null" json:"configuration_version"`
	NacosInstanceID      uint      `gorm:"not null;index:idx_deploy_hist_nacos_instance_id" json:"nacos_instance_id"`
	DeploymentType       string    `gorm:"type:varchar(50);default:'FULL'" json:"deployment_type"`               // e.g., "GRAY", "FULL"
	Status               string    `gorm:"type:varchar(50);not null;index:idx_deploy_hist_status" json:"status"` // e.g., "SUCCESS", "FAILURE", "IN_PROGRESS"
	Message              string    `gorm:"type:text" json:"message,omitempty"`                                   // Optional, for error messages or details from Nacos
	DeployedBy           string    `json:"deployed_by,omitempty"`                                                // Identifier of the user/system deploying it
	DeployedAt           time.Time `json:"deployed_at"`

	// Optional: Eager load related data if needed in responses, though often kept separate
	// Configuration   Configuration `gorm:"foreignKey:ConfigurationID" json:"-"` // Avoid cyclic dependencies if Configuration also has DeploymentHistory
	// NacosInstance NacosInstance `gorm:"foreignKey:NacosInstanceID" json:"-"`
}

// TableName specifies the table name for GORM.
func (DeploymentHistory) TableName() string {
	return "deployment_history"
}

// PublishRequest defines the structure for publishing requests (optional if no body params needed)
type PublishRequest struct {
	// For gray release, might include IPs, etc.
	// BetaIPs []string `json:"beta_ips,omitempty"`
	// For now, assuming no specific body params for simplicity with current Nacos SDK Go capabilities for basic publish
}
