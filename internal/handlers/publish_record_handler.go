package handlers

import (
	"nacos-config-tool/internal/logger"
	"nacos-config-tool/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PublishRecordHandler handles API requests related to publish records.
type PublishRecordHandler struct {
	DB *gorm.DB
}

// NewPublishRecordHandler creates a new handler for publish records.
func NewPublishRecordHandler(db *gorm.DB) *PublishRecordHandler {
	return &PublishRecordHandler{DB: db}
}

// ListPublishRecordsQuery defines query parameters for listing publish records.
type ListPublishRecordsQuery struct {
	ConfigurationID uint   `form:"configurationId"`
	NacosInstanceID uint   `form:"nacosInstanceId"`
	Status          string `form:"status" binding:"omitempty,oneof=SUCCESS FAILED PENDING"`
}

// ListPublishRecords handles listing publish records.
// GET /api/publish-records
func (h *PublishRecordHandler) ListPublishRecords(c *gin.Context) {
	var queryParams ListPublishRecordsQuery
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		logger.Warn("Invalid query parameters for ListPublishRecords", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	var records []models.PublishRecord
	dbQuery := h.DB.Model(&models.PublishRecord{})

	if queryParams.ConfigurationID > 0 {
		dbQuery = dbQuery.Where("configuration_id = ?", queryParams.ConfigurationID)
	}
	if queryParams.NacosInstanceID > 0 {
		dbQuery = dbQuery.Where("nacos_instance_id = ?", queryParams.NacosInstanceID)
	}
	if queryParams.Status != "" {
		dbQuery = dbQuery.Where("status = ?", queryParams.Status)
	}

	// Preload Configuration and NacosInstance details for context
	// Clear sensitive info like passwords from preloaded data
	if err := dbQuery.Preload("Configuration.NacosInstance").Preload("NacosInstance").Order("published_at desc").Find(&records).Error; err != nil {
		logger.Error("Failed to list publish records from DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list publish records: " + err.Error()})
		return
	}
	
	// Clear sensitive information from preloaded NacosInstance data
    for i := range records {
        if records[i].Configuration.ID > 0 && records[i].Configuration.NacosInstance.Password != "" {
             records[i].Configuration.NacosInstance.Password = ""
        }
		// Also clear password from the direct NacosInstance preload if it's separate
		if records[i].NacosInstance.Password != "" {
			records[i].NacosInstance.Password = ""
		}
    }


	c.JSON(http.StatusOK, records)
}


// GetPublishRecord retrieves a specific publish record by its ID.
// GET /api/publish-records/:id
func (h *PublishRecordHandler) GetPublishRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Warn("Invalid publish record ID format", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid publish record ID format"})
		return
	}

	var record models.PublishRecord
	// Preload related data and clear sensitive fields
	if err := h.DB.Preload("Configuration.NacosInstance").Preload("NacosInstance").First(&record, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("Publish record not found", zap.Uint64("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "Publish record not found"})
		} else {
			logger.Error("Failed to get publish record from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve publish record: " + err.Error()})
		}
		return
	}

    if record.Configuration.ID > 0 && record.Configuration.NacosInstance.Password != "" {
         record.Configuration.NacosInstance.Password = ""
    }
	if record.NacosInstance.Password != "" {
		record.NacosInstance.Password = ""
	}
	

	c.JSON(http.StatusOK, record)
}
