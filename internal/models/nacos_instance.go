package models

import "time"

// NacosInstance represents a configured Nacos server.
type NacosInstance struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	InstanceURL string    `gorm:"column:instance_url;type:varchar(255);not null;uniqueIndex:idx_instance_url_namespace,priority:1;comment:Nacos server address (e.g., http://localhost:8848 or https://nacos.example.com)"`
	NamespaceID string    `gorm:"column:namespace_id;type:varchar(255);not null;uniqueIndex:idx_instance_url_namespace,priority:2;default:'';comment:Namespace ID, empty for public"` // This is the default/management namespace for the instance if applicable
	Description string    `gorm:"column:description;type:varchar(500);comment:User-friendly description of the instance"`
	Username    string    `gorm:"column:username;type:varchar(100);comment:Username for Nacos authentication (optional)"`
	Password    string    `gorm:"column:password;type:varchar(255);comment:Password for Nacos authentication (optional, should be stored securely)"` // TODO: Encrypt this
	ContextPath string    `gorm:"column:context_path;type:varchar(100);default:'/nacos';comment:Nacos context path"`
	Scheme      string    `gorm:"column:scheme;type:varchar(10);default:'http';comment:http or https"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;index"`

	// Define relationships
	// OnDelete:RESTRICT will prevent deleting an instance if it has associated configurations or publish records.
	Configurations []Configuration `gorm:"foreignKey:NacosInstanceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	PublishRecords []PublishRecord `gorm:"foreignKey:NacosInstanceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (NacosInstance) TableName() string {
	return "nacos_instances"
}
