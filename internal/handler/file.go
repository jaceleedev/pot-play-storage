package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pot-play-storage/internal/model"
	"pot-play-storage/internal/service"
	"pot-play-storage/pkg/errors"
	"pot-play-storage/pkg/validator"
)

type FileHandler struct {
	service *service.StorageService
	logger  *zap.Logger
}

func NewFileHandler(svc *service.StorageService, logger *zap.Logger) *FileHandler {
	return &FileHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *FileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error("failed to get form file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload"})
		return
	}
	
	// Basic validation
	if file.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty file not allowed"})
		return
	}
	
	if file.Size > 100*1024*1024 { // 100MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
		return
	}
	
	reader, err := file.Open()
	if err != nil {
		h.logger.Error("failed to open uploaded file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to process file"})
		return
	}
	defer reader.Close()

	// MIME type detection
	mtype, err := mimetype.DetectReader(reader)
	if err != nil {
		h.logger.Error("failed to detect mime type", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file format"})
		return
	}
	
	_, err = reader.(io.Seeker).Seek(0, io.SeekStart)
	if err != nil {
		h.logger.Error("failed to reset file reader", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file processing error"})
		return
	}

	// Sanitize filename
	sanitizedName := validator.SanitizeFilename(file.Filename)
	
	header := &model.FileHeader{
		Name:        sanitizedName,
		Size:        file.Size,
		ContentType: mtype.String(),
	}
	
	// Comprehensive file validation
	if err := validator.ValidateFile(header); err != nil {
		h.logger.Error("file validation failed", zap.Error(err), zap.String("filename", file.Filename))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	result, err := h.service.Upload(c.Request.Context(), reader, header)
	if err != nil {
		h.logger.Error("upload service failed", zap.Error(err), zap.String("filename", file.Filename))
		
		// Use sanitized error message
		sanitizedMsg := errors.SanitizeError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizedMsg})
		return
	}
	
	c.JSON(http.StatusCreated, result)
}

func (h *FileHandler) Download(c *gin.Context) {
	id := c.Param("id")
	
	// Basic input validation
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file ID required"})
		return
	}
	
	reader, file, err := h.service.Download(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("download service failed", zap.Error(err), zap.String("file_id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer reader.Close()
	
	// Sanitize filename for Content-Disposition header
	safeName := fmt.Sprintf("file_%s", id)
	if file.Name != "" {
		// Remove potentially dangerous characters from filename
		safeName = file.Name
	}
	
	// Set security headers
	c.Header("Content-Type", file.GetContentType())
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", safeName))
	c.Header("Content-Length", fmt.Sprintf("%d", file.GetSize()))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("X-Content-Type-Options", "nosniff")
	
	// Stream the file
	_, err = io.Copy(c.Writer, reader)
	if err != nil {
		h.logger.Error("failed to stream file", zap.Error(err), zap.String("file_id", id))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}

func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	
	// Basic input validation
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file ID required"})
		return
	}
	
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete service failed", zap.Error(err), zap.String("file_id", id))
		
		// Use sanitized error message
		sanitizedMsg := errors.SanitizeError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizedMsg})
		return
	}
	
	c.Status(http.StatusNoContent)
}

func (h *FileHandler) List(c *gin.Context) {
	files, err := h.service.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list service failed", zap.Error(err))
		
		// Use sanitized error message
		sanitizedMsg := errors.SanitizeError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizedMsg})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"total": len(files),
	})
}