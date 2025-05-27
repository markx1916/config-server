package models

import "time"

// NacosInstance corresponds to the `nacos_instances` table in the database.
type NacosInstance struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	ServerAddr  string    `gorm:"type:varchar(255);not null" json:"server_addr"`
	NamespaceID string    `gorm:"type:varchar(255);default:null" json:"namespace_id"`
	Username    string    `gorm:"type:varchar(255);default:null" json:"username,omitempty"`
	Password    string    `gorm:"type:varchar(255);default:null" json:"-"` // Never send password in JSON responses
	CreatedAt   time.Time ` json:"created_at"`
	UpdatedAt   time.Time ` json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (NacosInstance) TableName() string {
	return "nacos_instances"
}

// CreateNacosInstanceRequest defines the structure for creating a new Nacos instance.
// Password is included here for creation.
type CreateNacosInstanceRequest struct {
	Name        string `json:"name" binding:"required"`
	ServerAddr  string `json:"server_addr" binding:"required"`
	NamespaceID string `json:"namespace_id"` // Optional
	Username    string `json:"username"`     // Optional
	Password    string `json:"password"`     // Optional
}

// UpdateNacosInstanceRequest defines the structure for updating an existing Nacos instance.
// All fields are optional.
type UpdateNacosInstanceRequest struct {
	Name        *string `json:"name,omitempty"`
	ServerAddr  *string `json:"server_addr,omitempty"`
	NamespaceID *string `json:"namespace_id,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"` // For updating password
}
