-- SQL Schema for Nacos Configuration Center
-- Compatible with MySQL 5.7

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- Table structure for nacos_instances
DROP TABLE IF EXISTS `nacos_instances`;
CREATE TABLE `nacos_instances` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary Key, Auto Increment',
  `name` VARCHAR(255) NOT NULL COMMENT 'e.g., "Dev Nacos", "Prod Nacos"',
  `server_addr` VARCHAR(255) NOT NULL COMMENT 'Nacos server address like "host:port"',
  `namespace_id` VARCHAR(255) DEFAULT NULL COMMENT 'Nacos namespace ID, can be empty for public',
  `username` VARCHAR(255) DEFAULT NULL COMMENT 'Optional username for Nacos authentication',
  `password` VARCHAR(255) DEFAULT NULL COMMENT 'Optional password for Nacos authentication (should be stored securely)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Timestamp of creation',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Timestamp of last update',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_server_addr_namespace` (`server_addr`, `namespace_id`) COMMENT 'Unique combination of server address and namespace'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Stores details of different Nacos server instances';

-- Table structure for configurations
DROP TABLE IF EXISTS `configurations`;
CREATE TABLE `configurations` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary Key, Auto Increment',
  `nacos_instance_id` INT UNSIGNED NOT NULL COMMENT 'Foreign Key referencing Nacos Instances',
  `data_id` VARCHAR(255) NOT NULL COMMENT 'Nacos Data ID',
  `group_name` VARCHAR(255) NOT NULL COMMENT 'Nacos Group',
  `content` TEXT NOT NULL COMMENT 'The actual configuration content',
  `format` VARCHAR(50) NOT NULL DEFAULT 'text' COMMENT 'e.g., "yaml", "json", "properties", "text"',
  `description` TEXT DEFAULT NULL COMMENT 'Optional description for the configuration',
  `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT 'Version number, incremented on each save',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Timestamp of creation',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Timestamp of last update',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_data_group` (`nacos_instance_id`, `data_id`, `group_name`) COMMENT 'A configuration is unique per instance, data_id, and group',
  KEY `idx_nacos_instance_id` (`nacos_instance_id`),
  KEY `idx_data_id` (`data_id`),
  KEY `idx_group_name` (`group_name`),
  CONSTRAINT `fk_configurations_nacos_instance`
    FOREIGN KEY (`nacos_instance_id`)
    REFERENCES `nacos_instances` (`id`)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Stores the actual configuration data';

-- Table structure for configuration_history
DROP TABLE IF EXISTS `configuration_history`;
CREATE TABLE `configuration_history` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary Key, Auto Increment',
  `configuration_id` INT UNSIGNED NOT NULL COMMENT 'Foreign Key referencing Configurations',
  `content` TEXT NOT NULL COMMENT 'The historical configuration content',
  `format` VARCHAR(50) NOT NULL COMMENT 'Historical format of the configuration',
  `version` INT UNSIGNED NOT NULL COMMENT 'The version number this history entry represents',
  `saved_by` VARCHAR(255) DEFAULT NULL COMMENT 'Identifier of the user/system saving it',
  `saved_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Timestamp of when this history entry was saved',
  PRIMARY KEY (`id`),
  KEY `idx_configuration_id` (`configuration_id`),
  KEY `idx_configuration_id_version` (`configuration_id`, `version`) COMMENT 'Query by configuration and version',
  CONSTRAINT `fk_history_configuration`
    FOREIGN KEY (`configuration_id`)
    REFERENCES `configurations` (`id`)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tracks changes to each configuration';

-- Table structure for deployment_history
DROP TABLE IF EXISTS `deployment_history`;
CREATE TABLE `deployment_history` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary Key, Auto Increment',
  `configuration_id` INT UNSIGNED NOT NULL COMMENT 'Foreign Key referencing Configurations (the config that was intended to be deployed)',
  `configuration_version` INT UNSIGNED NOT NULL COMMENT 'Version of the configuration deployed',
  `nacos_instance_id` INT UNSIGNED NOT NULL COMMENT 'Foreign Key referencing Nacos Instances (the target Nacos instance)',
  `deployment_type` VARCHAR(50) DEFAULT 'FULL' COMMENT 'e.g., "GRAY", "FULL"',
  `status` VARCHAR(50) NOT NULL COMMENT 'e.g., "SUCCESS", "FAILURE", "IN_PROGRESS"',
  `message` TEXT DEFAULT NULL COMMENT 'Optional, for error messages or details from Nacos',
  `deployed_by` VARCHAR(255) DEFAULT NULL COMMENT 'Identifier of the user/system deploying it',
  `deployed_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Timestamp of deployment attempt',
  PRIMARY KEY (`id`),
  KEY `idx_configuration_id_deployment` (`configuration_id`),
  KEY `idx_nacos_instance_id_deployment` (`nacos_instance_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deployed_at` (`deployed_at`),
  CONSTRAINT `fk_deployment_configuration`
    FOREIGN KEY (`configuration_id`)
    REFERENCES `configurations` (`id`)
    ON DELETE RESTRICT ON UPDATE CASCADE, -- Prevent deleting a config if it has deployment history, or handle it application-side
  CONSTRAINT `fk_deployment_nacos_instance`
    FOREIGN KEY (`nacos_instance_id`)
    REFERENCES `nacos_instances` (`id`)
    ON DELETE RESTRICT ON UPDATE CASCADE -- Prevent deleting a Nacos instance if it has deployment history
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Logs deployments of configurations to Nacos instances';

SET FOREIGN_KEY_CHECKS = 1;
