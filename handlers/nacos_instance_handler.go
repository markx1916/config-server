package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/nacos-config-center/db" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/models" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/utils"  // Adjust to your module path
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateNacosInstance godoc
// @Summary Create a new Nacos instance connection
// @Description Add a new Nacos server instance to the system
// @Tags NacosInstances
// @Accept  json
// @Produce  json
// @Param   instance_data body models.CreateNacosInstanceRequest true "Nacos Instance Data"
// @Success 201 {object} models.NacosInstance
// @Failure 400 {object} map[string]string "Invalid request payload or validation error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/instances [post]
func CreateNacosInstance(c *gin.Context) {
	var req models.CreateNacosInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Logger.Error("Failed to bind JSON for CreateNacosInstance", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	instance := models.NacosInstance{
		Name:        req.Name,
		ServerAddr:  req.ServerAddr,
		NamespaceID: req.NamespaceID,
		Username:    req.Username,
		// Note: Password should be hashed/encrypted in a real application before storing
		Password:    req.Password, // Storing plain for prototype, BAD PRACTICE
	}

	if err := db.DB.Create(&instance).Error; err != nil {
		utils.Logger.Error("Failed to create Nacos instance in DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Nacos instance: " + err.Error()})
		return
	}

	// Important: Do not return the password in the response, even if hashed.
	// The instance object here will have the password if it was set.
	// Create a new struct for response or clear the password field.
	instance.Password = "" 
	c.JSON(http.StatusCreated, instance)
}

// ListNacosInstances godoc
// @Summary List all Nacos instance connections
// @Description Get a list of all configured Nacos server instances
// @Tags NacosInstances
// @Produce  json
// @Success 200 {array} models.NacosInstance
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/instances [get]
func ListNacosInstances(c *gin.Context) {
	var instances []models.NacosInstance
	if err := db.DB.Find(&instances).Error; err != nil {
		utils.Logger.Error("Failed to list Nacos instances from DB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instances"})
		return
	}
	// Clear passwords before sending
	for i := range instances {
		instances[i].Password = ""
	}
	c.JSON(http.StatusOK, instances)
}

// GetNacosInstance godoc
// @Summary Get details of a specific Nacos instance
// @Description Retrieve a Nacos server instance by its ID
// @Tags NacosInstances
// @Produce  json
// @Param   id path int true "Nacos Instance ID"
// @Success 200 {object} models.NacosInstance
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Nacos instance not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/instances/{id} [get]
func GetNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var instance models.NacosInstance
	if err := db.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			utils.Logger.Error("Failed to get Nacos instance from DB", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance"})
		}
		return
	}
	instance.Password = "" // Clear password
	c.JSON(http.StatusOK, instance)
}

// UpdateNacosInstance godoc
// @Summary Update a Nacos instance connection
// @Description Modify an existing Nacos server instance by its ID
// @Tags NacosInstances
// @Accept  json
// @Produce  json
// @Param   id path int true "Nacos Instance ID"
// @Param   instance_data body models.UpdateNacosInstanceRequest true "Nacos Instance Data to Update"
// @Success 200 {object} models.NacosInstance
// @Failure 400 {object} map[string]string "Invalid ID format or request payload"
// @Failure 404 {object} map[string]string "Nacos instance not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/instances/{id} [put]
func UpdateNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var instance models.NacosInstance
	if err := db.DB.First(&instance, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found"})
		} else {
			utils.Logger.Error("Failed to find Nacos instance for update", zap.Uint64("id", id), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Nacos instance for update"})
		}
		return
	}

	var req models.UpdateNacosInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Logger.Error("Failed to bind JSON for UpdateNacosInstance", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Update fields if they are provided in the request
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.ServerAddr != nil {
		instance.ServerAddr = *req.ServerAddr
	}
	if req.NamespaceID != nil {
		instance.NamespaceID = *req.NamespaceID
	}
	if req.Username != nil {
		instance.Username = *req.Username
	}
	if req.Password != nil && *req.Password != "" { 
		// Handle password update: In a real app, hash this new password
		instance.Password = *req.Password // Storing plain for prototype
	}


	if err := db.DB.Save(&instance).Error; err != nil {
		utils.Logger.Error("Failed to update Nacos instance in DB", zap.Uint64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Nacos instance: " + err.Error()})
		return
	}
	instance.Password = "" // Clear password
	c.JSON(http.StatusOK, instance)
}

// DeleteNacosInstance godoc
// @Summary Delete a Nacos instance connection
// @Description Remove a Nacos server instance by its ID
// @Tags NacosInstances
// @Produce  json
// @Param   id path int true "Nacos Instance ID"
// @Success 204 "Successfully deleted"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Nacos instance not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/nacos/instances/{id} [delete]
func DeleteNacosInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// GORM's Delete with a primary key will set DeletedAt for soft delete if the model has gorm.DeletedAt
	// If not, it will perform a hard delete. Our model does not have gorm.DeletedAt.
	result := db.DB.Delete(&models.NacosInstance{}, uint(id))
	if result.Error != nil {
		utils.Logger.Error("Failed to delete Nacos instance from DB", zap.Uint64("id", id), zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete Nacos instance"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nacos instance not found or already deleted"})
		return
	}

	c.Status(http.StatusNoContent)
}
