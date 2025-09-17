package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/callmemars1/ytdlp-http/internal/utils"
	"github.com/callmemars1/ytdlp-http/internal/ytdlp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DownloadHandler struct {
	ytdlpService *ytdlp.Service
	logger       *zap.Logger
}

type DownloadRequest struct {
	URL     string             `json:"url" binding:"required"`
	Options *ytdlp.Options     `json:"options,omitempty"`
	Timeout int                `json:"timeout,omitempty"`
}

func NewDownloadHandler(ytdlpService *ytdlp.Service, logger *zap.Logger) *DownloadHandler {
	return &DownloadHandler{
		ytdlpService: ytdlpService,
		logger:       logger,
	}
}

func (h *DownloadHandler) SetupRoute(router gin.IRouter) {
	router.POST("/download", h.Handle)
}

func (h *DownloadHandler) Handle(c *gin.Context) {
	var req DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid download request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	h.logger.Info("Starting video download", 
		zap.String("url", req.URL), 
		zap.String("client_ip", c.ClientIP()))

	filePath, videoInfo, err := h.ytdlpService.DownloadVideo(ctx, req.URL, req.Options)
	if err != nil {
		h.logger.Error("Failed to download video", zap.Error(err), zap.String("url", req.URL))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "download_failed",
			"message": "Failed to download video: " + err.Error(),
		})
		return
	}
	defer func() {
		if cleanupErr := h.ytdlpService.CleanupFile(filePath); cleanupErr != nil {
			h.logger.Error("Failed to cleanup file", zap.Error(cleanupErr), zap.String("file", filePath))
		}
	}()

	reader, fileSize, err := h.ytdlpService.GetVideoReader(filePath)
	if err != nil {
		h.logger.Error("Failed to open video file for reading", zap.Error(err), zap.String("file", filePath))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "file_read_error",
			"message": "Failed to read downloaded file",
		})
		return
	}
	defer reader.Close()

	filename := h.generateDownloadFilename(filePath, videoInfo)
	contentType := utils.GetContentType(filePath)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	h.logger.Info("Streaming video file", 
		zap.String("filename", filename),
		zap.Int64("size", fileSize),
		zap.String("client_ip", c.ClientIP()))

	c.DataFromReader(http.StatusOK, fileSize, contentType, reader, nil)
}

func (h *DownloadHandler) generateDownloadFilename(filePath string, videoInfo *ytdlp.VideoInfo) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		ext = ".mp4"
	}

	var baseName string
	if videoInfo != nil && videoInfo.Title != "" {
		baseName = utils.SanitizeFilename(videoInfo.Title)
	} else {
		baseName = "video"
	}

	return fmt.Sprintf("%s%s", baseName, ext)
}