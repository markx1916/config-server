package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/nacos-config-center/db"    // Adjust to your module path
	"github.com/yourusername/nacos-config-center/models" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/utils"  // Adjust to your module path
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ListConfigurations godoc
// @Summary List configurations
// @Description Get a list of configurations based on filters
// @Tags Configurations
// @Produce  json
// @Param   nacos_instance_id query int true "Nacos Instance ID"
// @Param   data_id query string false "Data ID (partial match)"
// @Param   group query string false "Group (partial match)"
// @Success 200 {array} models.ConfigurationResponse
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs [get]
func ListConfigurations(c *gin.Context) {
	instanceIDStr := c.Query("nacos_instance_id")
	if instanceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nacos_instance_id is required"})
		return
	}
	instanceID, err := strconv.ParseUint(instanceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid nacos_instance_id format"})
		return
	}

	dataID := c.Query("data_id")
	group := c.Query("group")

	var configs []models.Configuration
	query := db.DB.Where("nacos_instance_id = ?", uint(instanceID))

	if dataID != "" {
		query = query.Where("data_id LIKE ?", "%"+dataID+"%")
	}
	if group != "" {
		query = query.Where("group_name LIKE ?", "%"+group+"%")
	}

	if err := query.Order("updated_at DESC").Find(&configs).Error; err != nil {
		utils.Logger.Error("Failed to list configurations from DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configurations"})
		return
	}

	var responses []models.ConfigurationResponse
	for _, cfg := range configs {
		responses = append(responses, models.ConfigurationResponse{
			ID:              cfg.ID,
			NacosInstanceID: cfg.NacosInstanceID,
			DataID:          cfg.DataID,
			GroupName:       cfg.GroupName,
			Content:         cfg.Content, // Consider if content should be omitted in list view for brevity
			Format:          cfg.Format,
			Description:     cfg.Description,
			Version:         cfg.Version,
			CreatedAt:       cfg.CreatedAt,
			UpdatedAt:       cfg.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// CreateConfiguration godoc
// @Summary Create a new configuration
// @Description Add a new configuration to the local database
// @Tags Configurations
// @Accept  json
// @Produce  json
// @Param   config_data body models.CreateConfigurationRequest true "Configuration Data"
// @Success 201 {object} models.ConfigurationResponse
// @Failure 400 {object} map[string]string "Invalid request payload or validation error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs [post]
func CreateConfiguration(c *gin.Context) {
	var req models.CreateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Logger.Error("Failed to bind JSON for CreateConfiguration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Check if Nacos Instance exists
	var nacosInstance models.NacosInstance
	if err := db.DB.First(&nacosInstance, req.NacosInstanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Nacos instance with ID %d not found", req.NacosInstanceID)})
			return
		}
		utils.Logger.Error("Failed to find Nacos instance", zap.Uint("id", req.NacosInstanceID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking Nacos instance"})
		return
	}

	// Check for existing configuration with same data_id, group_name for the instance
	var existingConfig models.Configuration
	err := db.DB.Where("nacos_instance_id = ? AND data_id = ? AND group_name = ?", req.NacosInstanceID, req.DataID, req.GroupName).First(&existingConfig).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Configuration with DataID '%s' and Group '%s' already exists for this Nacos instance.", req.DataID, req.GroupName)})
		return
	}
	if err != gorm.ErrRecordNotFound {
		utils.Logger.Error("Error checking for existing configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing configuration"})
		return
	}


	config := models.Configuration{
		NacosInstanceID: req.NacosInstanceID,
		DataID:          req.DataID,
		GroupName:       req.GroupName,
		Content:         req.Content,
		Format:          req.Format,
		Description:     req.Description,
		Version:         1, // Initial version
	}

	// Use a transaction to ensure atomicity
	tx := db.DB.Begin()
	if tx.Error != nil {
		utils.Logger.Error("Failed to begin transaction for CreateConfiguration", zap.Error(tx.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	if err := tx.Create(&config).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create configuration in DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration: " + err.Error()})
		return
	}

	historyEntry := models.ConfigurationHistory{
		ConfigurationID: config.ID,
		Content:         config.Content,
		Format:          config.Format,
		Version:         config.Version,
		SavedBy:         "system", // Placeholder for user identification
		SavedAt:         time.Now(),
	}

	if err := tx.Create(&historyEntry).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create configuration history in DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration history: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.Logger.Error("Failed to commit transaction for CreateConfiguration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	response := models.ConfigurationResponse{
		ID:              config.ID,
		NacosInstanceID: config.NacosInstanceID,
		DataID:          config.DataID,
		GroupName:       config.GroupName,
		Content:         config.Content,
		Format:          config.Format,
		Description:     config.Description,
		Version:         config.Version,
		CreatedAt:       config.CreatedAt,
		UpdatedAt:       config.UpdatedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// GetConfiguration godoc
// @Summary Get a specific configuration
// @Description Retrieve a configuration by its local database ID
// @Tags Configurations
// @Produce  json
// @Param   id path int true "Configuration ID"
// @Success 200 {object} models.ConfigurationResponse
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Configuration not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id} [get]
func GetConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.Configuration
	if err := db.DB.First(&config, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			utils.Logger.Error("Failed to get configuration from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration"})
		}
		return
	}

	response := models.ConfigurationResponse{
		ID:              config.ID,
		NacosInstanceID: config.NacosInstanceID,
		DataID:          config.DataID,
		GroupName:       config.GroupName,
		Content:         config.Content,
		Format:          config.Format,
		Description:     config.Description,
		Version:         config.Version,
		CreatedAt:       config.CreatedAt,
		UpdatedAt:       config.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// PublishConfigurationToNacos handles both GRAY and FULL releases.
// @Summary Publish a configuration to Nacos
// @Description Publishes the current version of the configuration from the local DB to the associated Nacos instance.
// @Tags Configurations
// @Accept  json
// @Produce  json
// @Param   id path int true "Configuration ID (local DB)"
// @Param   type path string true "Publish Type (gray or full)" Enums(gray, full)
// @Param   publish_data body models.PublishRequest false "Publish Data (e.g., beta_ips for gray release)"
// @Success 200 {object} map[string]interface{} "status: success/failure, message: details"
// @Failure 400 {object} map[string]string "Invalid ID, type, or payload"
// @Failure 404 {object} map[string]string "Configuration or Nacos instance not found"
// @Failure 500 {object} map[string]string "Internal server error or Nacos publish error"
// @Router /api/nacos/configs/{id}/publish/{type} [post]
func PublishConfigurationToNacos(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Configuration ID format"})
		return
	}

	publishType := strings.ToUpper(c.Param("type"))
	if publishType != "GRAY" && publishType != "FULL" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid publish type. Must be 'gray' or 'full'."})
		return
	}

	var publishReq models.PublishRequest
	// Attempt to bind JSON only if content type is JSON, otherwise ignore (e.g. for FULL with no body)
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&publishReq); err != nil {
			// Don't fail if body is empty but content-type is json, treat as no specific params
			// This check might need refinement based on strictness of body requirement for GRAY
			if err.Error() != "EOF" { // EOF means empty body
				utils.Logger.Error("Failed to bind JSON for PublishConfiguration", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
				return
			}
		}
	}
	// Extract betaIps from publishReq if available, for now assuming it's a simple string in the request
	// var betaIps string
	// if publishType == "GRAY" && len(publishReq.BetaIPs) > 0 {
	//  betaIps = strings.Join(publishReq.BetaIPs, ",")
	// }
	// For this example, we'll assume betaIps comes from a query or is not used if the SDK doesn't support it easily
	betaIpsQueryParam := c.Query("beta_ips") // Example: ?beta_ips=1.1.1.1,2.2.2.2

	var localConfig models.Configuration
	if err := db.DB.First(&localConfig, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found in local DB"})
		} else {
			utils.Logger.Error("Failed to get configuration from DB for publish", zap.Uint64("id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration for publishing"})
		}
		return
	}

	var nacosInstance models.NacosInstance
	if err := db.DB.First(&nacosInstance, localConfig.NacosInstanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated Nacos instance not found"})
		} else {
			utils.Logger.Error("Failed to get Nacos instance from DB for publish", zap.Uint("id", localConfig.NacosInstanceID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance details"})
		}
		return
	}

	// Record deployment attempt
	deployment := models.DeploymentHistory{
		ConfigurationID:      localConfig.ID,
		ConfigurationVersion: localConfig.Version,
		NacosInstanceID:      nacosInstance.ID,
		DeploymentType:       publishType,
		Status:               "IN_PROGRESS", // Initial status
		DeployedBy:           "api-user",    // Placeholder
		DeployedAt:           time.Now(),
	}
	if err := db.DB.Create(&deployment).Error; err != nil {
		utils.Logger.Error("Failed to create initial deployment history record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record deployment start"})
		return
	}

	success, nacosErr := services.PublishConfig(&nacosInstance, &localConfig, localConfig.Content, publishType, betaIpsQueryParam)

	if nacosErr != nil {
		deployment.Status = "FAILURE"
		deployment.Message = nacosErr.Error()
		if err := db.DB.Save(&deployment).Error; err != nil {
			utils.Logger.Error("Failed to update deployment history with Nacos failure", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failure", "message": "Failed to publish to Nacos: " + nacosErr.Error()})
		return
	}

	if !success { // Nacos SDK specific: success can be false without an error
		deployment.Status = "FAILURE"
		deployment.Message = "Nacos reported publish was not successful (SDK returned false)"
		if err := db.DB.Save(&deployment).Error; err != nil {
			utils.Logger.Error("Failed to update deployment history with Nacos non-success", zap.Error(err))
		}
		c.JSON(http.StatusOK, gin.H{"status": "failure", "message": deployment.Message})
		return
	}

	deployment.Status = "SUCCESS"
	deployment.Message = "Successfully published to Nacos."
	if err := db.DB.Save(&deployment).Error; err != nil {
		utils.Logger.Error("Failed to update deployment history with success", zap.Error(err))
		// Continue, as publish itself was successful
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Configuration published to Nacos successfully."})
}


// GetConfigurationHistoryList godoc
// @Summary List historical versions of a configuration
// @Description Get all historical versions for a given configuration ID from the local database
// @Tags Configurations
// @Produce  json
// @Param   id path int true "Configuration ID (local DB)"
// @Success 200 {array} models.ConfigurationHistory
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Configuration not found (to ensure history is for a valid config)"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id}/history [get]
func GetConfigurationHistoryList(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Configuration ID format"})
		return
	}

	// Optional: Check if the main configuration entry exists
	var config models.Configuration
	if err := db.DB.Select("id").First(&config, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			utils.Logger.Error("Error checking for configuration existence", zap.Uint64("id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validating configuration ID"})
		}
		return
	}

	var historyList []models.ConfigurationHistory
	if err := db.DB.Where("configuration_id = ?", uint(configID)).Order("version DESC").Find(&historyList).Error; err != nil {
		utils.Logger.Error("Failed to list configuration history from DB", zap.Uint64("config_id", configID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration history"})
		return
	}

	if len(historyList) == 0 {
		// Return empty list if no history, not an error
		c.JSON(http.StatusOK, []models.ConfigurationHistory{})
		return
	}

	c.JSON(http.StatusOK, historyList)
}

// RollbackConfiguration godoc
// @Summary Rollback a configuration to a historical version
// @Description Updates the current configuration content to a specified historical version's content. Does not auto-publish.
// @Tags Configurations
// @Produce  json
// @Param   id path int true "Configuration ID (local DB)"
// @Param   history_id path int true "Configuration History ID (local DB `configuration_history.id`)"
// @Success 200 {object} models.ConfigurationResponse "The updated configuration reflecting the rollback"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Configuration or History entry not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id}/rollback/{history_id} [post]
func RollbackConfiguration(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Configuration ID format"})
		return
	}

	historyIDStr := c.Param("history_id")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid History ID format"})
		return
	}

	var currentConfig models.Configuration
	if err := db.DB.First(&currentConfig, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Current configuration not found"})
		} else {
			utils.Logger.Error("Failed to get current configuration for rollback", zap.Uint64("id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve current configuration"})
		}
		return
	}

	var historyEntry models.ConfigurationHistory
	if err := db.DB.Where("id = ? AND configuration_id = ?", uint(historyID), uint(configID)).First(&historyEntry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("History entry with ID %d not found for configuration ID %d", uint(historyID), uint(configID))})
		} else {
			utils.Logger.Error("Failed to get history entry for rollback", zap.Uint64("history_id", historyID), zap.Uint64("config_id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve history entry"})
		}
		return
	}

	// Begin transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		utils.Logger.Error("Failed to begin transaction for RollbackConfiguration", zap.Error(tx.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Update current configuration
	currentConfig.Content = historyEntry.Content
	currentConfig.Format = historyEntry.Format
	currentConfig.Version++ // New version for this rollback state
	// Description could also be rolled back if stored in history and desired. Assuming it's not for now or part of content.

	if err := tx.Save(&currentConfig).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to save rolled-back configuration", zap.Uint64("id", configID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save rolled-back configuration"})
		return
	}

	// Create new history entry for this rollback action
	newHistoryForRollback := models.ConfigurationHistory{
		ConfigurationID: currentConfig.ID,
		Content:         currentConfig.Content,
		Format:          currentConfig.Format,
		Version:         currentConfig.Version,
		SavedBy:         "system-rollback", // Placeholder for user
		SavedAt:         time.Now(),
	}
	if err := tx.Create(&newHistoryForRollback).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create new history entry for rollback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create history entry for rollback"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.Logger.Error("Failed to commit transaction for RollbackConfiguration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	response := models.ConfigurationResponse{
		ID:              currentConfig.ID,
		NacosInstanceID: currentConfig.NacosInstanceID,
		DataID:          currentConfig.DataID,
		GroupName:       current.GroupName,
		Content:         currentConfig.Content,
		Format:          currentConfig.Format,
		Description:     currentConfig.Description, // This would be the description before rollback unless also rolled back
		Version:         currentConfig.Version,
		CreatedAt:       currentConfig.CreatedAt,
		UpdatedAt:       currentConfig.UpdatedAt, // This will be updated by GORM
	}
	c.JSON(http.StatusOK, response)
}

// GetDeploymentHistoryList godoc
// @Summary List deployment history for a configuration
// @Description Get all deployment history entries for a given configuration ID
// @Tags Configurations
// @Produce  json
// @Param   id path int true "Configuration ID (local DB)"
// @Success 200 {array} models.DeploymentHistory
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Configuration not found (to ensure history is for a valid config)"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id}/deployments [get]
func GetDeploymentHistoryList(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Configuration ID format"})
		return
	}

	// Optional: Check if the main configuration entry exists
	var config models.Configuration
	if err := db.DB.Select("id").First(&config, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			utils.Logger.Error("Error checking for configuration existence for deployments list", zap.Uint64("id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validating configuration ID"})
		}
		return
	}

	var deployments []models.DeploymentHistory
	if err := db.DB.Where("configuration_id = ?", uint(configID)).Order("deployed_at DESC").Find(&deployments).Error; err != nil {
		utils.Logger.Error("Failed to list deployment history from DB", zap.Uint64("config_id", configID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deployment history"})
		return
	}

	if len(deployments) == 0 {
		c.JSON(http.StatusOK, []models.DeploymentHistory{})
		return
	}

	c.JSON(http.StatusOK, deployments)
}

// UpdateConfiguration godoc
// @Summary Update a configuration
// @Description Modify an existing configuration in the local database
// @Tags Configurations
// @Accept  json
// @Produce  json
// @Param   id path int true "Configuration ID"
// @Param   config_data body models.UpdateConfigurationRequest true "Configuration Data to Update"
// @Success 200 {object} models.ConfigurationResponse
// @Failure 400 {object} map[string]string "Invalid ID format or request payload"
// @Failure 404 {object} map[string]string "Configuration not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id} [put]
func UpdateConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.Configuration
	if err := db.DB.First(&config, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			utils.Logger.Error("Failed to find configuration for update", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration for update"})
		}
		return
	}

	var req models.UpdateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Logger.Error("Failed to bind JSON for UpdateConfiguration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Track if any changes were made for versioning
	changed := false
	if req.Content != nil && config.Content != *req.Content {
		config.Content = *req.Content
		changed = true
	}
	if req.Format != nil && config.Format != *req.Format {
		config.Format = *req.Format
		changed = true
	}
	if req.Description != nil && config.Description != *req.Description {
		config.Description = *req.Description
		// Description change might not always warrant a version bump, depends on policy
		// For now, any tracked field change will bump version.
		changed = true
	}

	if !changed {
		// No actual changes to content, format, or description.
		// Return current config or a specific message.
		response := models.ConfigurationResponse{
			ID: config.ID, NacosInstanceID: config.NacosInstanceID, DataID: config.DataID, GroupName: config.GroupName,
			Content: config.Content, Format: config.Format, Description: config.Description, Version: config.Version,
			CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
		}
		c.JSON(http.StatusOK, response) // Or http.StatusNotModified if appropriate
		return
	}

	config.Version++ // Increment version

	// Use a transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		utils.Logger.Error("Failed to begin transaction for UpdateConfiguration", zap.Error(tx.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	if err := tx.Save(&config).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to update configuration in DB", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration: " + err.Error()})
		return
	}

	historyEntry := models.ConfigurationHistory{
		ConfigurationID: config.ID,
		Content:         config.Content,
		Format:          config.Format,
		Version:         config.Version,
		SavedBy:         "system-update", // Placeholder
		SavedAt:         time.Now(),
	}
	if err := tx.Create(&historyEntry).Error; err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to create configuration history for update", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration history for update: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.Logger.Error("Failed to commit transaction for UpdateConfiguration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Placeholder for diff generation:
	utils.Logger.Info("Diff generation needed here.", zap.Uint("config_id", config.ID))
	// Placeholder for Feishu notification:
	utils.Logger.Info("Feishu notification needed here.", zap.Uint("config_id", config.ID))


	response := models.ConfigurationResponse{
		ID:              config.ID,
		NacosInstanceID: config.NacosInstanceID,
		DataID:          config.DataID,
		GroupName:       config.GroupName,
		Content:         config.Content,
		Format:          config.Format,
		Description:     config.Description,
		Version:         config.Version,
		CreatedAt:       config.CreatedAt,
		UpdatedAt:       config.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// GetConfigurationDiff godoc
// @Summary Get diff for a configuration
// @Description Compare the current version of a configuration with a specified historical version
// @Tags Configurations
// @Produce  json
// @Param   id path int true "Configuration ID"
// @Param   compare_to_version query int false "Version to compare against (defaults to previous version)"
// @Success 200 {object} models.DiffResponse
// @Failure 400 {object} map[string]string "Invalid ID or version format"
// @Failure 404 {object} map[string]string "Configuration or version not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/configs/{id}/diff [get]
func GetConfigurationDiff(c *gin.Context) {
	idStr := c.Param("id")
	configID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Configuration ID format"})
		return
	}

	var currentConfig models.Configuration
	if err := db.DB.First(&currentConfig, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Current configuration not found"})
		} else {
			utils.Logger.Error("Failed to get current configuration from DB for diff", zap.Uint64("id", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve current configuration"})
		}
		return
	}

	compareToVersionStr := c.Query("compare_to_version")
	var compareToVersion uint
	if compareToVersionStr == "" {
		if currentConfig.Version <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No previous version to compare against for version 1."})
			return
		}
		compareToVersion = currentConfig.Version - 1
	} else {
		parsedVersion, err := strconv.ParseUint(compareToVersionStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid compare_to_version format"})
			return
		}
		compareToVersion = uint(parsedVersion)
		if compareToVersion >= currentConfig.Version {
			c.JSON(http.StatusBadRequest, gin.H{"error": "compare_to_version must be less than the current version"})
			return
		}
	}

	var comparedVersionHistory models.ConfigurationHistory
	err = db.DB.Where("configuration_id = ? AND version = ?", configID, compareToVersion).First(&comparedVersionHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Historical version %d not found for this configuration", compareToVersion)})
		} else {
			utils.Logger.Error("Failed to get historical configuration from DB for diff", zap.Uint64("id", configID), zap.Uint("version", compareToVersion), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve historical configuration version"})
		}
		return
	}

	// Placeholder for actual diff logic (e.g., using a library like diff-match-patch)
	// For now, just return both contents.
	diffOutput := fmt.Sprintf("--- Version %d\n%s\n+++ Current Version %d\n%s",
		comparedVersionHistory.Version,
		comparedVersionHistory.Content,
		currentConfig.Version,
		currentConfig.Content,
	)
	if strings.EqualFold(currentConfig.Content, comparedVersionHistory.Content) {
		diffOutput = "Contents are identical."
	}


	response := models.DiffResponse{
		CurrentVersionContent:  currentConfig.Content,
		ComparedVersion:        comparedVersionHistory.Version,
		ComparedVersionContent: comparedVersionHistory.Content,
		DiffOutput:             diffOutput, // Placeholder
	}

	c.JSON(http.StatusOK, response)
}
