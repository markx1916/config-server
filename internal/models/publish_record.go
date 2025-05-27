package models

import "time"

// PublishRecord logs attempts to publish configurations to Nacos.
type PublishRecord struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	ConfigurationID uint      `gorm:"column:configuration_id;not null;index:idx_publish_config_id;comment:Foreign key to configurations table"`
	NacosInstanceID uint      `gorm:"column:nacos_instance_id;not null;index:idx_publish_instance_id;comment:Foreign key to nacos_instances table (denormalized for easier querying)"`
	PublishType     string    `gorm:"column:publish_type;type:varchar(50);comment:Type of publish (e.g., 'FULL', 'GRAYSCALE')"` // Consider ENUM type if DB supports
	Status          string    `gorm:"column:status;type:varchar(50);index:idx_publish_status;comment:Status of the publish attempt (e.g., 'SUCCESS', 'FAILED', 'PENDING')"` // Consider ENUM type
	Details         string    `gorm:"column:details;type:text;comment:Details of the publish operation, including any error messages"`
	Operator        string    `gorm:"column:operator;type:varchar(100);comment:User or system process that initiated the publish"`
	PublishedAt     time.Time `gorm:"column:published_at;autoCreateTime;index"` // Timestamp of when the publish was initiated/recorded

	// Define relationships
	// OnDelete:RESTRICT will prevent deleting a configuration or Nacos instance if they have associated publish records.
	Configuration Configuration `gorm:"foreignKey:ConfigurationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	NacosInstance NacosInstance `gorm:"foreignKey:NacosInstanceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (PublishRecord) TableName() string {
	return "publish_records"
}
