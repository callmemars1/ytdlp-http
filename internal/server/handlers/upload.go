package handlers

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/callmemars1/ytdlp-http/internal/s3"
	"github.com/callmemars1/ytdlp-http/internal/utils"
	"github.com/callmemars1/ytdlp-http/internal/ytdlp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UploadHandler struct {
	ytdlpService *ytdlp.Service
	s3Service    *s3.Service
	logger       *zap.Logger
}

type UploadRequest struct {
	URL     string         `json:"url" binding:"required"`
	S3Key   string         `json:"s3_key" binding:"required"`
	Options *ytdlp.Options `json:"options,omitempty"`
	Timeout int            `json:"timeout,omitempty"`
}

type UploadResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Result  *s3.UploadResult    `json:"result,omitempty"`
	Error   string              `json:"error,omitempty"`
}

func NewUploadHandler(ytdlpService *ytdlp.Service, s3Service *s3.Service, logger *zap.Logger) *UploadHandler {
	return &UploadHandler{
		ytdlpService: ytdlpService,
		s3Service:    s3Service,
		logger:       logger,
	}
}

func (h *UploadHandler) SetupRoute(router gin.IRouter) {
	router.POST("/upload", h.Handle)
}

func (h *UploadHandler) Handle(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid upload request", zap.Error(err))
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	h.logger.Info("Starting video download and S3 upload", 
		zap.String("url", req.URL),
		zap.String("s3_key", req.S3Key),
		zap.String("client_ip", c.ClientIP()))

	filePath, videoInfo, err := h.ytdlpService.DownloadVideo(ctx, req.URL, req.Options)
	if err != nil {
		h.logger.Error("Failed to download video", zap.Error(err), zap.String("url", req.URL))
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Error:   "Failed to download video: " + err.Error(),
		})
		return
	}
	defer func() {
		if cleanupErr := h.ytdlpService.CleanupFile(filePath); cleanupErr != nil {
			h.logger.Error("Failed to cleanup file", zap.Error(cleanupErr), zap.String("file", filePath))
		}
	}()

	uniqueKey := h.generateUniqueS3Key(req.S3Key, filePath)

	uploadResult, err := h.s3Service.UploadVideoWithMetadata(ctx, filePath, uniqueKey, videoInfo)
	if err != nil {
		h.logger.Error("Failed to upload to S3", zap.Error(err), 
			zap.String("file", filePath),
			zap.String("s3_key", uniqueKey))
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Error:   "Failed to upload to S3: " + err.Error(),
		})
		return
	}

	h.logger.Info("Video uploaded successfully to S3", 
		zap.String("video_key", uploadResult.VideoUpload.Key),
		zap.String("metadata_key", uploadResult.MetadataUpload.Key),
		zap.Int64("total_size", uploadResult.TotalSize),
		zap.String("client_ip", c.ClientIP()))

	c.JSON(http.StatusOK, UploadResponse{
		Success: true,
		Message: "Video uploaded successfully to S3-compatible storage",
		Result:  uploadResult,
	})
}

func (h *UploadHandler) generateUniqueS3Key(requestedKey, filePath string) string {
	ext := filepath.Ext(filePath)
	
	if filepath.Ext(requestedKey) == "" {
		requestedKey = requestedKey + ext
	}
	
	safeName := utils.SanitizeFilename(requestedKey)
	return utils.GenerateUniqueKey(safeName)
}