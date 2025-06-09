package handlers

import (
	"fmt"
	"nacos-config-tool/internal/config"
	"nacos-config-tool/internal/logger"
	"nacos-config-tool/internal/models"
	"nacos-config-tool/pkg/nacosclient"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PublishHandler handles API requests related to publishing configurations.
type PublishHandler struct {
	DB                 *gorm.DB
	NacosClientMgr     *nacosclient.NacosClientManager
	FeishuWebhookURL   string
	DefaultNacosConfig config.NacosConfig
}

// NewPublishHandler creates a new handler for publishing operations.
func NewPublishHandler(db *gorm.DB, ncm *nacosclient.NacosClientManager, appCfg config.Config) *PublishHandler {
	return &PublishHandler{
		DB:                 db,
		NacosClientMgr:     ncm,
		FeishuWebhookURL:   appCfg.FeishuWebook,
		DefaultNacosConfig: appCfg.Nacos,
	}
}

// PublishConfigurationRequest defines the request body for publishing a configuration.
type PublishConfigurationRequest struct {
	PublishType string `json:"publishType" binding:"required,oneof=FULL GRAYSCALE"` // FULL or GRAYSCALE
	Operator    string `json:"operator"`                                           // Optional: User performing the publish
	// For GRAYSCALE, additional parameters like BetaIPS might be needed.
	// BetaIPS     string `json:"betaIps,omitempty"` // Comma-separated IPs for grayscale publish
}

// PublishConfiguration handles publishing a configuration to Nacos.
// POST /api/configurations/:id/publish
func (h *PublishHandler) PublishConfiguration(c *gin.Context) {
	configIDStr := c.Param("id")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid configuration ID format for publish", zap.String("id", configIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID format"})
		return
	}

	var req PublishConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request body for PublishConfiguration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	operator := "api_user" // Default operator
	if req.Operator != "" {
		operator = req.Operator
	}

	var localConfig models.Configuration
	if err := h.DB.Preload("NacosInstance").First(&localConfig, uint(configID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Configuration not found for publish", zap.Uint64("configId", configID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		} else {
			logger.Error("Failed to get configuration for publish from DB", zap.Uint64("configId", configID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration: " + err.Error()})
		}
		return
	}

	if localConfig.NacosInstance.ID == 0 {
	    logger.Warn("Nacos instance details not found for configuration during publish", zap.Uint64("configId", configID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nacos instance details not found for this configuration."})
		return
	}

	// Create PublishRecord with PENDING status
	publishRecord := models.PublishRecord{
		ConfigurationID: localConfig.ID,
		NacosInstanceID: localConfig.NacosInstanceID,
		PublishType:     req.PublishType,
		Status:          "PENDING",
		Operator:        operator,
		Details:         "Publish operation initiated.",
	}
	if err := h.DB.Create(&publishRecord).Error; err != nil {
		logger.Error("Failed to create initial publish record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create publish record: " + err.Error()})
		return
	}

	configClient, _, err := h.NacosClientMgr.GetClients(localConfig.NacosInstance)
	if err != nil {
		logger.Error("Failed to get Nacos client for publish", zap.Uint("configId", localConfig.ID), zap.Error(err))
		publishRecord.Status = "FAILED"
		publishRecord.Details = "Failed to get Nacos client: " + err.Error()
		h.DB.Save(&publishRecord) // Update record with failure
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Nacos client: " + err.Error()})
		return
	}

	nacosParams := vo.ConfigParam{
		DataId:    localConfig.DataID,
		Group:     localConfig.Group,
		Content:   localConfig.Content,
		Type:      vo.ConfigType(localConfig.ContentType), // Nacos SDK uses vo.ConfigType
		Namespace: localConfig.NamespaceID,
		// TODO: For GRAYSCALE, need to handle BetaIps if Nacos SDK supports it directly in PublishConfig or via other means.
		// AppName: if applicable for grayscale or tags
	}
	// if req.PublishType == "GRAYSCALE" && req.BetaIPS != "" {
	//    // Nacos SDK vo.ConfigParam doesn't directly have BetaIPS.
	//    // This might need a custom approach or check SDK for advanced publish options.
	//    // For now, we log a warning if grayscale is chosen but betaIps handling isn't fully implemented.
	//    logger.Warn("Grayscale publish selected, but BetaIPS handling might be limited by SDK's PublishConfig params.", zap.String("betaIps", req.BetaIPS))
	// }


	success, err := h.NacosClientMgr.PublishConfig(configClient, nacosParams)
	if err != nil {
		publishRecord.Status = "FAILED"
		publishRecord.Details = "Failed to publish to Nacos: " + err.Error()
		logger.Error("Nacos publish operation failed", zap.Error(err), zap.Uint("publishRecordId", publishRecord.ID))
	} else if !success {
		publishRecord.Status = "FAILED"
		publishRecord.Details = "Nacos reported publish as unsuccessful."
		logger.Warn("Nacos reported publish as unsuccessful", zap.Uint("publishRecordId", publishRecord.ID))
	} else {
		publishRecord.Status = "SUCCESS"
		publishRecord.Details = "Successfully published to Nacos."
		logger.Info("Successfully published configuration to Nacos", zap.Uint("publishRecordId", publishRecord.ID), zap.String("dataId", localConfig.DataID))

		// Update LastSync info on the configuration
		localConfig.LastSyncTime = func() *time.Time { t := time.Now(); return &t }()
		localConfig.LastSyncVersion = localConfig.MD5 // Assuming successful publish means local MD5 is now Nacos version
		if err := h.DB.Save(&localConfig).Error; err != nil {
		    // Log error but don't fail the whole operation as publish to Nacos was successful
            logger.Error("Failed to update LastSyncTime/Version on local config after successful Nacos publish", zap.Error(err), zap.Uint("configId", localConfig.ID))
        }

	}

	publishRecord.PublishedAt = time.Now() // Record actual publish attempt time more accurately
	if err := h.DB.Save(&publishRecord).Error; err != nil {
		logger.Error("Failed to update publish record status", zap.Error(err), zap.Uint("publishRecordId", publishRecord.ID))
		// Don't overwrite client response if publish itself was successful
		if publishRecord.Status != "SUCCESS" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update publish record: " + err.Error()})
			return
		}
	}

	// Mock Feishu notification for successful publish
	if publishRecord.Status == "SUCCESS" && h.FeishuWebhookURL != "" {
		logger.Info("Mock Feishu Notification: Configuration published to Nacos",
			zap.Uint("configId", localConfig.ID),
			zap.String("dataId", localConfig.DataID),
			zap.String("group", localConfig.Group),
			zap.String("publishType", req.PublishType),
			zap.String("operator", operator),
		)
	} else if publishRecord.Status == "SUCCESS" {
		logger.Info("Feishu webhook URL not configured. Skipping notification for successful publish.", zap.Uint("configId", localConfig.ID))
	}


	if publishRecord.Status == "SUCCESS" {
		c.JSON(http.StatusOK, gin.H{"message": "Configuration published successfully", "publishRecord": publishRecord})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": publishRecord.Details, "publishRecord": publishRecord})
	}
}
