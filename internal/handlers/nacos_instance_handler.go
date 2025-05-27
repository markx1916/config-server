package handlers

import (
	"fmt"
	"nacos-config-tool/internal/config"
	"nacos-config-tool/internal/database"
	"nacos-config-tool/internal/logger"
	"nacos-config-tool/internal/models"
	"nacos-config-tool/pkg/nacosclient"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NacosInstanceHandler handles API requests related to Nacos instances.
type NacosInstanceHandler struct {
	DB                 *gorm.DB
	NacosClientMgr     *nacosclient.NacosClientManager
	DefaultNacosConfig config.NacosConfig
}

// NewNacosInstanceHandler creates a new handler for Nacos instances.
func NewNacosInstanceHandler(db *gorm.DB, ncm *nacosclient.NacosClientManager, appCfg config.Config) *NacosInstanceHandler {
	return &NacosInstanceHandler{
		DB:                 db,
		NacosClientMgr:     ncm,
		DefaultNacosConfig: appCfg.Nacos, // Store default Nacos config from app's main config
	}
}

// CreateNacosInstanceRequest defines the request body for creating a Nacos instance.
type CreateNacosInstanceRequest struct {
	InstanceURL string `json:"instanceUrl" binding:"required,url"`
	NamespaceID string `json:"namespaceId"` // Optional, defaults to "" (public)
	Description string `json:"description"`
	Username    string `json:"username"`    // Optional
	Password    string `json:"password"`    // Optional
	Scheme      string `json:"scheme" binding:"omitempty,oneof=http https"` // Optional, defaults to http
	ContextPath string `json:"contextPath"` // Optional, defaults to /nacos
}

// CreateNacosInstance handles the creation of a new Nacos instance.
// POST /api/nacos-instances
func (h *NacosInstanceHandler) CreateNacosInstance(c *gin.Context) {
	var req CreateNacosInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request body for CreateNacosInstance", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	instance := models.NacosInstance{
		InstanceURL: req.InstanceURL,
		NamespaceID: req.NamespaceID, // Defaults to empty string if not provided, which is GORM's default
		Description: req.Description,
		Username:    req.Username,
		Password:    req.Password, // TODO: Add encryption/hashing
		Scheme:      req.Scheme,
		ContextPath: req.ContextPath,
	}
	if instance.Scheme == "" {
		instance.Scheme = "http" // Default scheme
	}
	if instance.ContextPath == "" {
		instance.ContextPath = "/nacos" // Default context path
	}


	// Check for existing instance with the same URL and NamespaceID
	var existing models.NacosInstance
	if err := h.DB.Where("instance_url = ? AND namespace_id = ?", instance.InstanceURL, instance.NamespaceID).First(&existing).Error; err == nil {
		logger.Warn("Attempt to create a duplicate Nacos instance", zap.String("url", instance.InstanceURL), zap.String("namespace", instance.NamespaceID))
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Nacos instance with URL '%s' and NamespaceID '%s' already exists", instance.InstanceURL, instance.NamespaceID)})
		return
	} else if err != gorm.ErrRecordNotFound {
		logger.Error("Error checking for existing Nacos instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing instance: " + err.Error()})
		return
	}


	if err := h.DB.Create(&instance).Error; err != nil {
		logger.Error("Failed to create Nacos instance in DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Nacos instance: " + err.Error()})
		return
	}

	logger.Info("Successfully created Nacos instance", zap.Uint("id", instance.ID), zap.String("url", instance.InstanceURL))
	c.JSON(http.StatusCreated, instance)
}

// ListNacosInstances handles listing all Nacos instances.
// GET /api/nacos-instances
func (h *NacosInstanceHandler) ListNacosInstances(c *gin.Context) {
	var instances []models.NacosInstance
	if err := h.DB.Find(&instances).Error; err != nil {
		logger.Error("Failed to list Nacos instances from DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Nacos instances: " + err.Error()})
		return
	}
	// For security, clear passwords before sending to client
	for i := range instances {
		instances[i].Password = ""
	}
	c.JSON(http.StatusOK, instances)
}

// GetNacosInstance handles retrieving a specific Nacos instance by ID.
// GET /api/nacos-instances/:id
func (h *NacosInstanceHandler) GetNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid Nacos instance ID format", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	var instance models.NacosInstance
	if err := h.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Nacos instance not found", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			logger.Error("Failed to get Nacos instance from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance: " + err.Error()})
		}
		return
	}
	// For security, clear password before sending to client
	instance.Password = ""
	c.JSON(http.StatusOK, instance)
}

// UpdateNacosInstanceRequest defines the request body for updating a Nacos instance.
// Similar to Create, but fields are optional.
type UpdateNacosInstanceRequest struct {
	InstanceURL string `json:"instanceUrl,omitempty" binding:"omitempty,url"`
	NamespaceID string `json:"namespaceId,omitempty"`
	Description *string `json:"description,omitempty"` // Pointer to distinguish between empty string and not provided
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"` // If provided, it will be updated. Empty means no change to password.
	Scheme      string `json:"scheme,omitempty" binding:"omitempty,oneof=http https"`
	ContextPath string `json:"contextPath,omitempty"`
}


// UpdateNacosInstance handles updating an existing Nacos instance.
// PUT /api/nacos-instances/:id
func (h *NacosInstanceHandler) UpdateNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid Nacos instance ID format for update", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	var req UpdateNacosInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request body for UpdateNacosInstance", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	var instance models.NacosInstance
	if err := h.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Nacos instance not found for update", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			logger.Error("Failed to get Nacos instance for update from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance: " + err.Error()})
		}
		return
	}

	// Apply updates from request
	if req.InstanceURL != "" {
		instance.InstanceURL = req.InstanceURL
	}
	// Note: NamespaceID is part of a unique key with InstanceURL. Changing it might require careful consideration
	// For now, we allow it, but it could lead to conflicts if not handled properly by unique constraints or checks.
	// if req.NamespaceID != "" { // This check is problematic if user wants to set it to "" (public) from a non-empty value
	instance.NamespaceID = req.NamespaceID // Allow setting to empty string for public namespace
	// }

	if req.Description != nil {
		instance.Description = *req.Description
	}
	if req.Username != "" { // Allow updating username
		instance.Username = req.Username
	}
	if req.Password != "" { // Only update password if a new one is provided
		instance.Password = req.Password // TODO: Add encryption/hashing
	}
	if req.Scheme != "" {
		instance.Scheme = req.Scheme
	}
    if req.ContextPath != "" {
        instance.ContextPath = req.ContextPath
    } else if req.ContextPath == "" && c.ContentType() == "application/json" { // Check if it was explicitly set to empty
		var raw map[string]interface{}
		if c.ShouldBind(&raw) == nil {
			if _, ok := raw["contextPath"]; ok {
				instance.ContextPath = "" // Allow setting contextPath to empty if explicitly provided
			}
		}
	}


	// Check for potential conflict if InstanceURL or NamespaceID changed
	var conflictingInstance models.NacosInstance
	query := h.DB.Where("instance_url = ? AND namespace_id = ? AND id != ?", instance.InstanceURL, instance.NamespaceID, instance.ID).First(&conflictingInstance)
	if query.Error == nil {
		logger.Warn("Update would result in a duplicate Nacos instance", zap.String("url", instance.InstanceURL), zap.String("namespace", instance.NamespaceID))
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("An instance with URL '%s' and NamespaceID '%s' already exists.", instance.InstanceURL, instance.NamespaceID)})
		return
	} else if query.Error != gorm.ErrRecordNotFound {
		logger.Error("Error checking for conflicting Nacos instance during update", zap.Error(query.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for conflicts: " + query.Error.Error()})
		return
	}


	if err := h.DB.Save(&instance).Error; err != nil {
		logger.Error("Failed to update Nacos instance in DB", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Nacos instance: " + err.Error()})
		return
	}

	logger.Info("Successfully updated Nacos instance", zap.Uint("id", instance.ID))
	instance.Password = "" // Clear password before sending response
	c.JSON(http.StatusOK, instance)
}

// DeleteNacosInstance handles deleting a Nacos instance.
// DELETE /api/nacos-instances/:id
func (h *NacosInstanceHandler) DeleteNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid Nacos instance ID format for delete", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	// Check if there are related configurations or publish records
	var configCount int64
	h.DB.Model(&models.Configuration{}).Where("nacos_instance_id = ?", uint(id)).Count(&configCount)
	if configCount > 0 {
		errMsg := fmt.Sprintf("Cannot delete Nacos instance: %d configurations are associated with it. Please delete them first.", configCount)
		logger.Warn(errMsg, zap.Uint64("id", id))
		c.JSON(http.StatusConflict, gin.H{"error": errMsg})
		return
	}
	// Publish records have RESTRICT on NacosInstanceID, so GORM should prevent deletion if they exist.
	// We can add an explicit check if needed, similar to configurations.

	if err := h.DB.Delete(&models.NacosInstance{}, uint(id)).Error; err != nil {
		// Check if the error is because of existing publish records due to RESTRICT constraint
		// This check is a bit generic; specific DB error codes might be better
		if database.IsForeignKeyConstraintError(err) { // Assuming IsForeignKeyConstraintError is a helper you'd write
			logger.Warn("Cannot delete Nacos instance due to existing publish records", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete Nacos instance: it has associated publish records. Please remove them first."})
			return
		}
		logger.Error("Failed to delete Nacos instance from DB", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete Nacos instance: " + err.Error()})
		return
	}
	
	// GORM's Delete with a primary key value will set DeletedAt field if the model has gorm.DeletedAt
    // If it's a hard delete, then RowsAffected can be checked.
    // For soft delete, we assume success if no error.
    // To be more precise, one might check h.DB.RowsAffected after the delete operation.

	logger.Info("Successfully deleted Nacos instance (or marked as deleted)", zap.Uint64("id", id))
	c.JSON(http.StatusOK, gin.H{"message": "Nacos instance deleted successfully"})
}


// TestNacosInstanceConnection handles testing connectivity to a Nacos instance.
// POST /api/nacos-instances/:id/test-connection
func (h *NacosInstanceHandler) TestNacosInstanceConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid Nacos instance ID format for test connection", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	var instance models.NacosInstance
	if err := h.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Nacos instance not found for test connection", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			logger.Error("Failed to get Nacos instance for test connection from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance: " + err.Error()})
		}
		return
	}

	// Use the NacosClientManager to test connection
	// The TestNacosConnection method in NacosClientManager uses the instance details
	// and the default Nacos config (e.g., for timeout) passed during NCM initialization.
	err = h.NacosClientMgr.TestNacosConnection(instance)
	if err != nil {
		logger.Error("Nacos connection test failed", zap.Uint64("id", id), zap.String("url", instance.InstanceURL), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Nacos connection test failed: " + err.Error(),
			"message": "Connection failed",
			"status":  "failed",
		})
		return
	}

	logger.Info("Nacos connection test successful", zap.Uint64("id", id), zap.String("url", instance.InstanceURL))
	c.JSON(http.StatusOK, gin.H{
		"message": "Nacos connection test successful",
		"status":  "successful",
	})
}

// ListNacosInstanceNamespaces lists namespaces for a given Nacos instance.
// GET /api/nacos-instances/:id/namespaces
func (h *NacosInstanceHandler) ListNacosInstanceNamespaces(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid Nacos instance ID format for listing namespaces", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	var instance models.NacosInstance
	if err := h.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Nacos instance not found for listing namespaces", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			logger.Error("Failed to get Nacos instance for listing namespaces from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance: " + err.Error()})
		}
		return
	}

	_, namingClient, err := h.NacosClientMgr.GetClients(instance)
	if err != nil {
		logger.Error("Failed to get Nacos clients for listing namespaces", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Nacos clients: " + err.Error()})
		return
	}

	namespaces, err := h.NacosClientMgr.GetAllNamespaces(namingClient)
	if err != nil {
		logger.Error("Failed to list Nacos namespaces", zap.Uint64("id", id), zap.String("url", instance.InstanceURL), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Nacos namespaces: " + err.Error()})
		return
	}

	// The Nacos SDK's model.Namespace has Namespace and NamespaceShowName.
	// We can return this directly or map it to a simpler struct if needed.
	c.JSON(http.StatusOK, namespaces)
}
