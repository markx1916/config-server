package models

import "time"

// Configuration represents a managed configuration.
type Configuration struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	NacosInstanceID uint      `gorm:"column:nacos_instance_id;not null;uniqueIndex:idx_config_unique,priority:3;comment:Foreign key to nacos_instances table"`
	DataID          string    `gorm:"column:data_id;type:varchar(255);not null;uniqueIndex:idx_config_unique,priority:1;comment:Nacos Data ID"`
	Group           string    `gorm:"column:group_name;type:varchar(128);not null;default:'DEFAULT_GROUP';uniqueIndex:idx_config_unique,priority:2;comment:Nacos Group"`
	// NamespaceID here refers to the specific Nacos namespace for THIS DataID/Group on the NacosInstance.
	// It can be different from NacosInstance.NamespaceID (which might be a default/management namespace for the instance itself).
	NamespaceID     string    `gorm:"column:namespace_id;type:varchar(255);not null;default:'';uniqueIndex:idx_config_unique,priority:4;comment:Nacos Namespace ID for this specific configuration, empty for public"`
	Content         string    `gorm:"column:content;type:longtext;not null;comment:Configuration content"`
	ContentType     string    `gorm:"column:content_type;type:varchar(50);default:'text';comment:Type of content (e.g., text, json, yaml, properties)"`
	Description     string    `gorm:"column:description;type:varchar(500);comment:User-friendly description"`
	MD5             string    `gorm:"column:md5_sum;type:varchar(32);comment:MD5 hash of the content for quick comparison"`
	LastSyncVersion string    `gorm:"column:last_sync_version;type:varchar(64);comment:Identifier (e.g., MD5 or version number) of the last successfully synced version with Nacos"`
	LastSyncTime    *time.Time `gorm:"column:last_sync_time;comment:Timestamp of the last successful sync with Nacos"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime;index"`

	// Define relationships
	// The unique index `idx_config_unique` covers: DataID, Group, NacosInstanceID, NamespaceID.
	NacosInstance      NacosInstance          `gorm:"foreignKey:NacosInstanceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"` // Restrict deleting instance if configs exist for it
	// OnDelete:CASCADE means if a Configuration is deleted, its history and publish records are also deleted.
	History            []ConfigurationHistory `gorm:"foreignKey:ConfigurationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	PublishRecords     []PublishRecord        `gorm:"foreignKey:ConfigurationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Configuration) TableName() string {
	return "configurations"
}
