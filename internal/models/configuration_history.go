package models

import "time"

// ConfigurationHistory stores historical versions of configurations.
type ConfigurationHistory struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	ConfigurationID uint      `gorm:"column:configuration_id;not null;index:idx_history_config_id_created_at,priority:1;comment:Foreign key to configurations table"`
	Content         string    `gorm:"column:content;type:longtext;not null;comment:Configuration content at this version"`
	ContentType     string    `gorm:"column:content_type;type:varchar(50);default:'text';comment:Type of content (e.g., text, json, yaml, properties)"`
	MD5             string    `gorm:"column:md5_sum;type:varchar(32);comment:MD5 hash of the content"`
	ChangeSource    string    `gorm:"column:change_source;type:varchar(50);comment:Source of the change (e.g., 'user_edit', 'nacos_sync', 'rollback')"` // Consider ENUM/VARCHAR
	Operator        string    `gorm:"column:operator;type:varchar(100);comment:User or system process that made the change"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;index:idx_history_config_id_created_at,priority:2"` // Combined index for querying and sorting

	// Define relationship
	// OnDelete:CASCADE means if the parent Configuration is deleted, its history entries are also deleted.
	Configuration   Configuration `gorm:"foreignKey:ConfigurationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (ConfigurationHistory) TableName() string {
	return "configuration_history"
}
