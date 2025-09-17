package s3

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/callmemars1/ytdlp-http/internal/configurations"
	"github.com/callmemars1/ytdlp-http/internal/utils"
	"github.com/callmemars1/ytdlp-http/internal/ytdlp"
	"go.uber.org/zap"
)

type Service struct {
	client *s3.Client
	config *configurations.S3Config
	logger *zap.Logger
}

type UploadResult struct {
	VideoUpload    *FileUploadResult `json:"video_upload"`
	MetadataUpload *FileUploadResult `json:"metadata_upload"`
	TotalSize      int64             `json:"total_size"`
	UploadedAt     time.Time         `json:"uploaded_at"`
}

type FileUploadResult struct {
	Key         string `json:"key"`
	Bucket      string `json:"bucket"`
	Location    string `json:"location"`
	ETag        string `json:"etag"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	MD5Hash     string `json:"md5_hash"`
}

func NewService(cfg *configurations.S3Config, logger *zap.Logger) (*Service, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			SigningRegion:     cfg.Region,
			HostnameImmutable: true,
		}, nil
	})

	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load S3-compatible config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &Service{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

func (s *Service) UploadVideoWithMetadata(ctx context.Context, filePath, key string, videoInfo *ytdlp.VideoInfo) (*UploadResult, error) {
	s.logger.Info("Starting video and metadata upload", zap.String("file", filePath), zap.String("key", key))

	videoResult, err := s.uploadFile(ctx, filePath, key)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	metadataKey := s.getMetadataKey(key)
	metadataResult, err := s.uploadMetadata(ctx, metadataKey, videoInfo, filepath.Base(filePath))
	if err != nil {
		s.logger.Error("Failed to upload metadata, cleaning up video", zap.Error(err))
		if delErr := s.DeleteFile(ctx, key); delErr != nil {
			s.logger.Error("Failed to cleanup video after metadata upload failure", zap.Error(delErr))
		}
		return nil, fmt.Errorf("failed to upload metadata: %w", err)
	}

	result := &UploadResult{
		VideoUpload:    videoResult,
		MetadataUpload: metadataResult,
		TotalSize:      videoResult.Size + metadataResult.Size,
		UploadedAt:     time.Now(),
	}

	s.logger.Info("Video and metadata uploaded successfully", 
		zap.String("video_key", key),
		zap.String("metadata_key", metadataKey),
		zap.Int64("total_size", result.TotalSize))

	return result, nil
}

func (s *Service) uploadFile(ctx context.Context, filePath, key string) (*FileUploadResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	contentType := utils.GetContentType(filePath)
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate MD5 hash: %w", err)
	}
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(stat.Size()),
		Metadata: map[string]string{
			"original-filename": filepath.Base(filePath),
			"upload-timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3-compatible storage: %w", err)
	}

	location := fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, key)

	return &FileUploadResult{
		Key:         key,
		Bucket:      s.config.Bucket,
		Location:    location,
		ETag:        strings.Trim(aws.ToString(result.ETag), `"`),
		Size:        stat.Size(),
		ContentType: contentType,
		MD5Hash:     md5Hash,
	}, nil
}

func (s *Service) uploadMetadata(ctx context.Context, key string, videoInfo *ytdlp.VideoInfo, originalFilename string) (*FileUploadResult, error) {
	metadata := map[string]interface{}{
		"original_filename": originalFilename,
		"upload_timestamp":  time.Now().Format(time.RFC3339),
	}

	if videoInfo != nil {
		metadata["video_info"] = videoInfo
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	hash := md5.New()
	hash.Write(jsonData)
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(jsonData),
		ContentType:   aws.String("application/json"),
		ContentLength: aws.Int64(int64(len(jsonData))),
		Metadata: map[string]string{
			"content-type":     "application/json",
			"upload-timestamp": time.Now().Format(time.RFC3339),
		},
	}

	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload metadata to S3-compatible storage: %w", err)
	}

	location := fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, key)

	return &FileUploadResult{
		Key:         key,
		Bucket:      s.config.Bucket,
		Location:    location,
		ETag:        strings.Trim(aws.ToString(result.ETag), `"`),
		Size:        int64(len(jsonData)),
		ContentType: "application/json",
		MD5Hash:     md5Hash,
	}, nil
}

func (s *Service) DeleteFile(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		s.logger.Error("Failed to delete object from S3-compatible storage", zap.Error(err), zap.String("key", key))
		return fmt.Errorf("failed to delete object: %w", err)
	}

	s.logger.Info("File deleted from S3-compatible storage", zap.String("key", key))
	return nil
}

func (s *Service) DeleteVideoAndMetadata(ctx context.Context, videoKey string) error {
	metadataKey := s.getMetadataKey(videoKey)
	
	var errors []error
	
	if err := s.DeleteFile(ctx, videoKey); err != nil {
		errors = append(errors, fmt.Errorf("failed to delete video: %w", err))
	}
	
	if err := s.DeleteFile(ctx, metadataKey); err != nil {
		errors = append(errors, fmt.Errorf("failed to delete metadata: %w", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("deletion errors: %v", errors)
	}
	
	return nil
}

func (s *Service) getMetadataKey(videoKey string) string {
	ext := filepath.Ext(videoKey)
	base := strings.TrimSuffix(videoKey, ext)
	return base + ".json"
}

