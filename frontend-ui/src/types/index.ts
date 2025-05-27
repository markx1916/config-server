// src/types/index.ts

// Represents a Nacos server instance as stored in our database
export interface NacosInstance {
  ID: number; // Changed from id to ID to match GORM model
  InstanceURL: string;
  NamespaceID: string; // Default/management namespace for the instance itself
  Description: string;
  Username?: string;
  // Password is not sent to frontend
  Scheme: string;
  ContextPath: string;
  CreatedAt: string; // ISO Date string
  UpdatedAt: string; // ISO Date string
}

// For creating/updating Nacos instances
export interface NacosInstancePayload {
  instanceUrl: string; // Matches backend CreateNacosInstanceRequest
  namespaceId?: string;
  description?: string;
  username?: string;
  password?: string; // Only for create/update, not stored long-term on frontend
  scheme?: string;
  contextPath?: string;
}

// Represents a Nacos namespace as returned by the Nacos SDK/API
export interface NacosNamespace {
  namespace: string; // Typically the ID
  namespaceShowName: string;
  type: number; // 0: public, 1: private, 2: custom
  quota?: number;
  configCount?: number;
}


// Represents a Configuration entry as stored in our database
export interface Configuration {
  ID: number; // Changed from id to ID
  NacosInstanceID: number;
  DataID: string;
  Group: string; // Renamed from groupName to Group to match GORM
  NamespaceID: string; // Specific Nacos namespace for this configuration
  Content: string;
  ContentType: string; // e.g., text, json, yaml, properties
  Description: string;
  MD5: string;
  LastSyncVersion?: string;
  LastSyncTime?: string | null; // ISO Date string or null
  CreatedAt: string; // ISO Date string
  UpdatedAt: string; // ISO Date string
  NacosInstance?: NacosInstance; // Optional, if preloaded
}

// For creating configurations
export interface ConfigurationPayload {
  nacosInstanceId: number;
  dataId: string;
  groupName: string; // Matches backend CreateConfigurationRequest
  namespaceId?: string;
  content: string;
  contentType?: string;
  description?: string;
}

// For updating configurations
export interface ConfigurationUpdatePayload {
  content?: string;
  contentType?: string;
  description?: string;
  operator?: string;
}


// Represents a Configuration History entry
export interface ConfigurationHistory {
  ID: number; // Changed from id to ID
  ConfigurationID: number;
  Content: string;
  ContentType: string;
  MD5: string;
  ChangeSource: string;
  Operator: string;
  CreatedAt: string; // ISO Date string
  Configuration?: Configuration; // Optional, if preloaded
}

// Represents a Publish Record
export interface PublishRecord {
  ID: number; // Changed from id to ID
  ConfigurationID: number;
  NacosInstanceID: number;
  PublishType: string; // e.g., 'FULL', 'GRAYSCALE'
  Status: string; // e.g., 'SUCCESS', 'FAILED', 'PENDING'
  Details: string;
  Operator: string;
  PublishedAt: string; // ISO Date string
  Configuration?: Configuration; // Optional, if preloaded
  NacosInstance?: NacosInstance; // Optional, if preloaded
}

// For publishing a configuration
export interface PublishPayload {
  publishType: 'FULL' | 'GRAYSCALE';
  operator?: string;
  // betaIps?: string; // For grayscale, if supported
}

// For rollback request
export interface RollbackPayload {
    operator?: string;
}


// Generic API error structure (if your backend sends errors in a specific format)
export interface ApiError {
  error: string;
  details?: any;
}

// For pagination if implemented
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
