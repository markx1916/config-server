package handlers

import (
	"crypto/md5"
	"fmt"
	"nacos-config-tool/internal/config"
	"nacos-config-tool/internal/database"
	"nacos-config-tool/internal/logger"
	"nacos-config-tool/internal/models"
	"nacos-config-tool/pkg/nacosclient"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	// For diff, could use "github.com/sergi/go-diff/diffmatchpatch"
	// "github.com/kylelemons/godebug/diff" // Another option for diff
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConfigurationHandler handles API requests related to configurations.
type ConfigurationHandler struct {
	DB                 *gorm.DB
	NacosClientMgr     *nacosclient.NacosClientManager
	FeishuWebhookURL   string // For mock notifications
	DefaultNacosConfig config.NacosConfig
}

// NewConfigurationHandler creates a new handler for configurations.
func NewConfigurationHandler(db *gorm.DB, ncm *nacosclient.NacosClientManager, appCfg config.Config) *ConfigurationHandler {
	return &ConfigurationHandler{
		DB:                 db,
		NacosClientMgr:     ncm,
		FeishuWebhookURL:   appCfg.FeishuWebook, // Corrected from FeishuWebook to FeishuWebhookURL
		DefaultNacosConfig: appCfg.Nacos,
	}
}

// CreateConfigurationRequest defines the request body for creating a configuration.
type CreateConfigurationRequest struct {
	NacosInstanceID uint   `json:"nacosInstanceId" binding:"required"`
	DataID          string `json:"dataId" binding:"required"`
	Group           string `json:"groupName" binding:"required"` // Nacos group, "groupName" for consistency with schema
	NamespaceID     string `json:"namespaceId"`                // Nacos Namespace ID for this config, optional (defaults to public)
	Content         string `json:"content" binding:"required"`
	ContentType     string `json:"contentType" binding:"omitempty,oneof=text json xml yaml html properties"`
	Description     string `json:"description"`
}

// CreateConfiguration handles creating a new configuration entry in the local DB.
// POST /api/configurations
func (h *ConfigurationHandler) CreateConfiguration(c *gin.Context) {
	var req CreateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request body for CreateConfiguration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "text" // Default content type
	}
	
	md5Sum := fmt.Sprintf("%x", md5.Sum([]byte(req.Content)))

	configEntry := models.Configuration{
		NacosInstanceID: req.NacosInstanceID,
		DataID:          req.DataID,
		Group:           req.Group,
		NamespaceID:     req.NamespaceID, // Default is empty string for public namespace
		Content:         req.Content,
		ContentType:     contentType,
		Description:     req.Description,
		MD5:             md5Sum,
	}

	// Validate NacosInstanceID exists
	var nacosInstance models.NacosInstance
	if err := h.DB.First(&nacosInstance, req.NacosInstanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Nacos instance not found for creating configuration", zap.Uint("nacosInstanceId", req.NacosInstanceID))
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("NacosInstance with ID %d not found", req.NacosInstanceID)})
			return
		}
		logger.Error("Error fetching Nacos instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validating Nacos instance: " + err.Error()})
		return
	}
	
	// Check for existing configuration
	var existing models.Configuration
	err := h.DB.Where("nacos_instance_id = ? AND data_id = ? AND group_name = ? AND namespace_id = ?",
		req.NacosInstanceID, req.DataID, req.Group, req.NamespaceID).First(&existing).Error
	if err == nil {
		errMsg := fmt.Sprintf("Configuration with DataID '%s', Group '%s', NacosInstanceID '%d', and NamespaceID '%s' already exists.", req.DataID, req.Group, req.NacosInstanceID, req.NamespaceID)
		logger.Warn(errMsg)
		c.JSON(http.StatusConflict, gin.H{"error": errMsg})
		return
	}
	if err != gorm.ErrRecordNotFound {
		logger.Error("Error checking for existing configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing configuration: " + err.Error()})
		return
	}


	// Create in DB and also create an initial history entry
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&configEntry).Error; err != nil {
			return fmt.Errorf("failed to create configuration: %w", err)
		}

		historyEntry := models.ConfigurationHistory{
			ConfigurationID: configEntry.ID,
			Content:         configEntry.Content,
			ContentType:     configEntry.ContentType,
			MD5:             configEntry.MD5,
			ChangeSource:    "initial_creation",
			Operator:        "system", // Or get from authenticated user context later
		}
		if err := tx.Create(&historyEntry).Error; err != nil {
			return fmt.Errorf("failed to create initial configuration history: %w", err)
		}
		return nil
	})

	if err != nil {
		logger.Error("Failed to create configuration and initial history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration: " + err.Error()})
		return
	}

	logger.Info("Successfully created configuration", zap.Uint("id", configEntry.ID), zap.String("dataId", configEntry.DataID), zap.String("group", configEntry.Group))
	c.JSON(http.StatusCreated, configEntry)
}

// ListConfigurationsQuery defines query parameters for listing configurations.
type ListConfigurationsQuery struct {
	NacosInstanceID uint   `form:"nacosInstanceId"`
	NamespaceID     string `form:"namespaceId"`
	DataID          string `form:"dataId"`
	Group           string `form:"group"`
}

// ListConfigurations handles listing configurations from the local DB.
// GET /api/configurations
func (h *ConfigurationHandler) ListConfigurations(c *gin.Context) {
	var queryParams ListConfigurationsQuery
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		logger.Warn("Invalid query parameters for ListConfigurations", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	var configurations []models.Configuration
	dbQuery := h.DB.Model(&models.Configuration{})

	if queryParams.NacosInstanceID > 0 {
		dbQuery = dbQuery.Where("nacos_instance_id = ?", queryParams.NacosInstanceID)
	}
	if queryParams.NamespaceID != "" { // Allow searching for specific namespace, including "" for public
		dbQuery = dbQuery.Where("namespace_id = ?", queryParams.NamespaceID)
	}
	if queryParams.DataID != "" {
		dbQuery = dbQuery.Where("data_id LIKE ?", "%"+queryParams.DataID+"%")
	}
	if queryParams.Group != "" {
		dbQuery = dbQuery.Where("group_name LIKE ?", "%"+queryParams.Group+"%")
	}

	// Preload NacosInstance details for context, but clear password
	if err := dbQuery.Preload("NacosInstance").Order("id desc").Find(&configurations).Error; err != nil {
		logger.Error("Failed to list configurations from DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list configurations: " + err.Error()})
		return
	}
	
	for i := range configurations {
        if configurations[i].NacosInstance.Password != "" {
            configurations[i].NacosInstance.Password = "" // Clear password
        }
    }

	c.JSON(http.StatusOK, configurations)
}

// GetConfiguration handles retrieving a specific configuration by ID.
// GET /api/configurations/:id
func (h *ConfigurationHandler) GetConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	var configEntry models.Configuration
	// Preload NacosInstance and clear password
	if err := h.DB.Preload("NacosInstance").First(&configEntry, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Configuration not found", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			logger.Error("Failed to get configuration from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration: " + err.Error()})
		}
		return
	}
	if configEntry.NacosInstance.Password != "" {
        configEntry.NacosInstance.Password = "" // Clear password
    }

	c.JSON(http.StatusOK, configEntry)
}

// UpdateConfigurationRequest defines the request body for updating a configuration.
type UpdateConfigurationRequest struct {
	Content     *string `json:"content"` // Pointer to distinguish between empty and not provided
	ContentType *string `json:"contentType" binding:"omitempty,oneof=text json xml yaml html properties"`
	Description *string `json:"description"`
	Operator    string  `json:"operator"` // Optional: User performing the update
}

// UpdateConfiguration handles updating a configuration in the local DB.
// PUT /api/configurations/:id
func (h *ConfigurationHandler) UpdateConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for update", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	var req UpdateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request body for UpdateConfiguration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	var configEntry models.Configuration
	if err := h.DB.First(&configEntry, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Configuration not found for update", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			logger.Error("Failed to get configuration for update from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration: " + err.Error()})
		}
		return
	}

	contentChanged := false
	if req.Content != nil && configEntry.Content != *req.Content {
		configEntry.Content = *req.Content
		configEntry.MD5 = fmt.Sprintf("%x", md5.Sum([]byte(configEntry.Content)))
		contentChanged = true
	}
	if req.ContentType != nil && configEntry.ContentType != *req.ContentType {
		configEntry.ContentType = *req.ContentType
		contentChanged = true // Or consider this a metadata change, not necessarily a full content change for history
	}
	if req.Description != nil {
		configEntry.Description = *req.Description
	}
	
	operator := "api_user" // Default operator, can be enhanced with auth
	if req.Operator != "" {
		operator = req.Operator
	}


	// Save updated config and create history entry in a transaction
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&configEntry).Error; err != nil {
			return fmt.Errorf("failed to save updated configuration: %w", err)
		}

		if contentChanged { // Only create history if content or type actually changed
			historyEntry := models.ConfigurationHistory{
				ConfigurationID: configEntry.ID,
				Content:         configEntry.Content,
				ContentType:     configEntry.ContentType,
				MD5:             configEntry.MD5,
				ChangeSource:    "user_edit",
				Operator:        operator,
			}
			if err := tx.Create(&historyEntry).Error; err != nil {
				return fmt.Errorf("failed to create configuration history: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("Failed to update configuration and history", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration: " + err.Error()})
		return
	}

	// Mock Feishu notification
	if h.FeishuWebhookURL != "" {
		logger.Info("Mock Feishu Notification: Configuration updated",
			zap.Uint("configId", configEntry.ID),
			zap.String("dataId", configEntry.DataID),
			zap.String("group", configEntry.Group),
			zap.String("operator", operator),
			// In a real scenario, construct a proper message payload for Feishu
		)
	} else {
		logger.Info("Feishu webhook URL not configured. Skipping notification for configuration update.", zap.Uint("configId", configEntry.ID))
	}


	logger.Info("Successfully updated configuration", zap.Uint("id", configEntry.ID))
	c.JSON(http.StatusOK, configEntry)
}

// GetConfigurationDiffWithNacos handles diffing local DB version with Nacos version.
// GET /api/configurations/:id/diff-nacos
func (h *ConfigurationHandler) GetConfigurationDiffWithNacos(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for diff", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	var localConfig models.Configuration
	if err := h.DB.Preload("NacosInstance").First(&localConfig, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Local configuration not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve local configuration: " + err.Error()})
		}
		return
	}
	
	if localConfig.NacosInstance.ID == 0 {
	    c.JSON(http.StatusBadRequest, gin.H{"error": "Nacos instance details not found for this configuration."})
		return
	}

	configClient, _, err := h.NacosClientMgr.GetClients(localConfig.NacosInstance)
	if err != nil {
		logger.Error("Failed to get Nacos client for diff", zap.Uint("configId", localConfig.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Nacos client: " + err.Error()})
		return
	}

	nacosContent, err := h.NacosClientMgr.GetConfig(configClient, vo.ConfigParam{
		DataId: localConfig.DataID,
		Group:  localConfig.Group,
		// NamespaceId for GetConfig should be the one associated with the config, not the NacosInstance's default.
		// The Nacos client itself is initialized with a default NamespaceId from NacosInstance,
		// but operations like GetConfig can specify a different one.
		// If localConfig.NamespaceID is empty, it implies the public namespace for that specific config.
		Namespace: localConfig.NamespaceID, 
	})

	if err != nil {
		// Nacos error could mean config not found in Nacos, or other issues
		logger.Warn("Failed to get config from Nacos for diff", zap.Uint("configId", localConfig.ID), zap.Error(err))
		// Distinguish "not found" from other errors if Nacos SDK provides typed errors
		if strings.Contains(err.Error(), "config data not exist") || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusOK, gin.H{
				"diff":         fmt.Sprintf("Local content:\n%s\n\nNacos: Configuration not found.", localConfig.Content),
				"localContent": localConfig.Content,
				"nacosContent": "",
				"isDifferent":  true, // Different because one exists and the other doesn't
				"nacosError":   "Configuration not found in Nacos.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config from Nacos: " + err.Error()})
		return
	}

	isDifferent := localConfig.Content != nacosContent
	diffResult := "Contents are identical."
	if isDifferent {
		// Simple diff: just show both contents.
		// For a real diff, use a library like github.com/sergi/go-diff/diffmatchpatch
		// dmp := diffmatchpatch.New()
		// diffs := dmp.DiffMain(localConfig.Content, nacosContent,  true) // Correct order for "local vs remote"
		// diffResult = dmp.DiffPrettyText(diffs)
		// Using a simpler text representation for now
		diffResult = fmt.Sprintf("--- Local DB\n+++ Nacos\n")
		// This is a placeholder for a proper diff algorithm.
		// A real diff library would show line-by-line changes.
		if localConfig.Content == nacosContent {
			diffResult += "Contents are identical."
		} else {
			diffResult += fmt.Sprintf("- %s\n+ %s", strings.ReplaceAll(localConfig.Content, "\n", "\n- "), strings.ReplaceAll(nacosContent, "\n", "\n+ "))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"diff":         diffResult,
		"localContent": localConfig.Content,
		"nacosContent": nacosContent,
		"isDifferent":  isDifferent,
	})
}

// FetchConfigurationFromNacos handles fetching config from Nacos and updating local DB.
// POST /api/configurations/:id/fetch-from-nacos
func (h *ConfigurationHandler) FetchConfigurationFromNacos(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for fetch", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}
	
	operator := c.Query("operator") // Optional: operator from query param
	if operator == "" {
		operator = "api_fetch"
	}

	var localConfig models.Configuration
	if err := h.DB.Preload("NacosInstance").First(&localConfig, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Local configuration not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve local configuration: " + err.Error()})
		}
		return
	}
	
	if localConfig.NacosInstance.ID == 0 {
	    c.JSON(http.StatusBadRequest, gin.H{"error": "Nacos instance details not found for this configuration."})
		return
	}

	configClient, _, err := h.NacosClientMgr.GetClients(localConfig.NacosInstance)
	if err != nil {
		logger.Error("Failed to get Nacos client for fetch", zap.Uint("configId", localConfig.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Nacos client: " + err.Error()})
		return
	}

	nacosContent, err := h.NacosClientMgr.GetConfig(configClient, vo.ConfigParam{
		DataId:    localConfig.DataID,
		Group:     localConfig.Group,
		Namespace: localConfig.NamespaceID,
	})
	if err != nil {
		logger.Warn("Failed to get config from Nacos for fetch", zap.Uint("configId", localConfig.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config from Nacos: " + err.Error()})
		return
	}

	if localConfig.Content == nacosContent {
		logger.Info("Content from Nacos is identical to local, no update needed.", zap.Uint("configId", localConfig.ID))
		c.JSON(http.StatusOK, gin.H{"message": "Content is identical, no update performed.", "configuration": localConfig})
		return
	}

	originalMD5 := localConfig.MD5
	localConfig.Content = nacosContent
	localConfig.MD5 = fmt.Sprintf("%x", md5.Sum([]byte(nacosContent)))
	localConfig.LastSyncTime = func() *time.Time { t := time.Now(); return &t }() // Update sync time
	// Assuming Nacos doesn't directly version like this, we might use its MD5 or a timestamp if available from Nacos headers
	localConfig.LastSyncVersion = localConfig.MD5 


	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&localConfig).Error; err != nil {
			return fmt.Errorf("failed to save updated configuration from Nacos: %w", err)
		}

		historyEntry := models.ConfigurationHistory{
			ConfigurationID: localConfig.ID,
			Content:         localConfig.Content,
			ContentType:     localConfig.ContentType, // ContentType might not be available from Nacos GetConfig, assume it's unchanged
			MD5:             localConfig.MD5,
			ChangeSource:    "nacos_sync_fetch",
			Operator:        operator, 
		}
		if err := tx.Create(&historyEntry).Error; err != nil {
			return fmt.Errorf("failed to create configuration history after Nacos fetch: %w", err)
		}
		return nil
	})

	if err != nil {
		logger.Error("Failed to update configuration from Nacos and create history", zap.Uint("configId", localConfig.ID), zap.Error(err), zap.String("originalMD5", originalMD5))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration from Nacos: " + err.Error()})
		return
	}
	
	// Mock Feishu notification
	if h.FeishuWebhookURL != "" {
		logger.Info("Mock Feishu Notification: Configuration fetched from Nacos and updated",
			zap.Uint("configId", localConfig.ID),
			zap.String("dataId", localConfig.DataID),
			zap.String("group", localConfig.Group),
			zap.String("operator", operator),
		)
	} else {
		logger.Info("Feishu webhook URL not configured. Skipping notification for Nacos fetch.", zap.Uint("configId", localConfig.ID))
	}


	logger.Info("Successfully fetched and updated configuration from Nacos", zap.Uint("configId", localConfig.ID))
	c.JSON(http.StatusOK, localConfig)
}

// ListConfigurationHistory lists history versions for a configuration.
// GET /api/configurations/:id/history
func (h *ConfigurationHandler) ListConfigurationHistory(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for history listing", zap.String("id", configIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	// Check if configuration exists
	var configEntry models.Configuration
	if err := h.DB.First(&configEntry, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Configuration not found for history listing", zap.Uint64("configId", configID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			logger.Error("Error fetching configuration for history", zap.Uint64("configId", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration details"})
		}
		return
	}

	var historyEntries []models.ConfigurationHistory
	if err := h.DB.Where("configuration_id = ?", uint(configID)).Order("created_at desc").Find(&historyEntries).Error; err != nil {
		logger.Error("Failed to list configuration history from DB", zap.Uint64("configId", configID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list configuration history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, historyEntries)
}

// GetConfigurationHistoryEntry gets a specific historical version's content.
// GET /api/configurations/history/:historyId
func (h *ConfigurationHandler) GetConfigurationHistoryEntry(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid history ID format", zap.String("id", historyIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID format"})
		return
	}

	var historyEntry models.ConfigurationHistory
	// Preload Configuration and its NacosInstance for context, clear password
	if err := h.DB.Preload("Configuration.NacosInstance").First(&historyEntry, uint(historyID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Configuration history entry not found", zap.Uint64("historyId", historyID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration history entry not found"})
		} else {
			logger.Error("Failed to get configuration history entry from DB", zap.Uint64("historyId", historyID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration history entry: " + err.Error()})
		}
		return
	}
	
    if historyEntry.Configuration.NacosInstance.Password != "" {
        historyEntry.Configuration.NacosInstance.Password = "" // Clear password
    }


	c.JSON(http.StatusOK, historyEntry)
}

// RollbackConfigurationRequest defines the request body for rollback.
type RollbackConfigurationRequest struct {
	Operator string `json:"operator"` // Optional: User performing the rollback
}

// RollbackConfiguration sets the content of the configuration to a specific history version.
// POST /api/configurations/:id/rollback/:historyId
func (h *ConfigurationHandler) RollbackConfiguration(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for rollback", zap.String("id", configIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	historyIDStr := c.Param("historyId")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid history ID format for rollback", zap.String("id", historyIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID format"})
		return
	}
	
	var req RollbackConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body, operator is optional
		if err.Error() != "EOF" { // EOF means empty body, which is fine here
			logger.Warn("Invalid request body for RollbackConfiguration", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}
	}
	
	operator := "api_user_rollback" // Default operator
	if req.Operator != "" {
		operator = req.Operator
	}


	var configToUpdate models.Configuration
	if err := h.DB.First(&configToUpdate, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration to rollback not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration to rollback: " + err.Error()})
		}
		return
	}

	var historyToRollbackFrom models.ConfigurationHistory
	if err := h.DB.Where("id = ? AND configuration_id = ?", uint(historyID), uint(configID)).First(&historyToRollbackFrom).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "History entry to rollback from not found or does not belong to the specified configuration"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve history entry: " + err.Error()})
		}
		return
	}

	// Update current configuration with content from history
	configToUpdate.Content = historyToRollbackFrom.Content
	configToUpdate.ContentType = historyToRollbackFrom.ContentType // Also rollback type
	configToUpdate.MD5 = historyToRollbackFrom.MD5          // And MD5

	// Create a new history entry for this rollback action
	newHistoryForRollback := models.ConfigurationHistory{
		ConfigurationID: configToUpdate.ID,
		Content:         configToUpdate.Content,
		ContentType:     configToUpdate.ContentType,
		MD5:             configToUpdate.MD5,
		ChangeSource:    fmt.Sprintf("rollback_to_history_id_%d", historyToRollbackFrom.ID),
		Operator:        operator,
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&configToUpdate).Error; err != nil {
			return fmt.Errorf("failed to save rolled-back configuration: %w", err)
		}
		if err := tx.Create(&newHistoryForRollback).Error; err != nil {
			return fmt.Errorf("failed to create new history entry for rollback: %w", err)
		}
		return nil
	})

	if err != nil {
		logger.Error("Rollback transaction failed", zap.Uint64("configId", configID), zap.Uint64("historyId", historyID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rollback failed: " + err.Error()})
		return
	}
	
	// Mock Feishu notification for rollback
	if h.FeishuWebhookURL != "" {
		logger.Info("Mock Feishu Notification: Configuration rolled back",
			zap.Uint("configId", configToUpdate.ID),
			zap.String("dataId", configToUpdate.DataID),
			zap.String("group", configToUpdate.Group),
			zap.Uint("rolledBackToHistoryId", historyToRollbackFrom.ID),
			zap.String("operator", operator),
		)
	} else {
		logger.Info("Feishu webhook URL not configured. Skipping notification for rollback.", zap.Uint("configId", configToUpdate.ID))
	}


	logger.Info("Successfully rolled back configuration", zap.Uint("configId", configToUpdate.ID), zap.Uint("historyId", historyToRollbackFrom.ID))
	c.JSON(http.StatusOK, gin.H{
		"message":              "Configuration rolled back successfully. Please review and publish if needed.",
		"updatedConfiguration": configToUpdate,
	})
}
