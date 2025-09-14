package handlers

import (
	"net/http"

	"docker-gui-backend/internal/database"
	"docker-gui-backend/internal/docker"
	"docker-gui-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
	dockerClient *docker.Client
	db           *database.DB
}

func NewImageHandler(dockerClient *docker.Client, db *database.DB) *ImageHandler {
	return &ImageHandler{
		dockerClient: dockerClient,
		db:           db,
	}
}

func (h *ImageHandler) ListImages(c *gin.Context) {
	images, err := h.dockerClient.ListImages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, images)
}

func (h *ImageHandler) PullImage(c *gin.Context) {
	var req models.PullImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ImageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "imageName is required"})
		return
	}

	err := h.dockerClient.PullImage(c.Request.Context(), req.ImageName)
	if err != nil {
		h.db.LogContainerAction("system", req.ImageName, "pull_image_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction("system", req.ImageName, "pull_image", "docker-gui", "Image pulled successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Image pulled successfully"})
}

func (h *ImageHandler) RemoveImage(c *gin.Context) {
	imageID := c.Param("id")
	force := c.DefaultQuery("force", "false") == "true"

	err := h.dockerClient.RemoveImage(c.Request.Context(), imageID, force)
	if err != nil {
		h.db.LogContainerAction("system", imageID, "remove_image_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction("system", imageID, "remove_image", "docker-gui", "Image removed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Image removed successfully"})
}

func (h *ImageHandler) PruneImages(c *gin.Context) {
	err := h.dockerClient.PruneImages(c.Request.Context())
	if err != nil {
		h.db.LogContainerAction("system", "images", "prune_images_failed", "docker-gui", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.LogContainerAction("system", "images", "prune_images", "docker-gui", "Images pruned successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Images pruned successfully"})
}