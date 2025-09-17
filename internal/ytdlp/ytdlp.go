package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	logger *zap.Logger
	mu     sync.Mutex
	tmpDir string
}

type VideoInfo struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Duration    float64 `json:"duration"`
	Uploader    string  `json:"uploader"`
	UploadDate  string  `json:"upload_date"`
	ViewCount   int64   `json:"view_count"`
	Format      string  `json:"format"`
	Filename    string  `json:"filename"`
	Filesize    int64   `json:"filesize"`
	URL         string  `json:"url"`
	Thumbnail   string  `json:"thumbnail"`
	Description string  `json:"description"`
}

type Options struct {
	Format      string            `json:"format,omitempty"`
	AudioOnly   bool              `json:"audio_only,omitempty"`
	VideoOnly   bool              `json:"video_only,omitempty"`
	Quality     string            `json:"quality,omitempty"`
	OutputPath  string            `json:"output_path,omitempty"`
	ExtraArgs   map[string]string `json:"extra_args,omitempty"`
	MaxFileSize string            `json:"max_file_size,omitempty"`
}

func NewService(logger *zap.Logger) *Service {
	tmpDir := filepath.Join(os.TempDir(), "ytdlp-downloads")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		logger.Fatal("Failed to create temp directory", zap.Error(err))
	}

	return &Service{
		logger: logger,
		tmpDir: tmpDir,
	}
}

func (s *Service) GetVideoInfo(ctx context.Context, url string) (*VideoInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Getting video info", zap.String("url", url))

	cmd := exec.CommandContext(ctx, "yt-dlp", 
		"--dump-json", 
		"--no-playlist", 
		url)
	
	output, err := cmd.Output()
	if err != nil {
		s.logger.Error("Failed to get video info", zap.Error(err), zap.String("url", url))
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		s.logger.Error("Failed to parse video info", zap.Error(err))
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	s.logger.Info("Video info retrieved", zap.String("title", info.Title), zap.String("id", info.ID))
	return &info, nil
}

func (s *Service) DownloadVideo(ctx context.Context, url string, options *Options) (string, *VideoInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting video download", zap.String("url", url))

	uniqueDir := filepath.Join(s.tmpDir, fmt.Sprintf("download_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(uniqueDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	args := []string{
		"--no-playlist",
		"--output", filepath.Join(uniqueDir, "%(title)s.%(ext)s"),
		"--write-info-json",
	}

	if options != nil {
		if options.Format != "" {
			args = append(args, "--format", options.Format)
		}
		if options.AudioOnly {
			args = append(args, "--extract-audio", "--audio-format", "mp3")
		}
		if options.VideoOnly {
			args = append(args, "--format", "best[height<=720]")
		}
		if options.Quality != "" {
			args = append(args, "--format", fmt.Sprintf("best[height<=%s]", options.Quality))
		}
		if options.MaxFileSize != "" {
			args = append(args, "--max-filesize", options.MaxFileSize)
		}
		
		for key, value := range options.ExtraArgs {
			args = append(args, fmt.Sprintf("--%s", key), value)
		}
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	if err := cmd.Run(); err != nil {
		s.logger.Error("Failed to download video", zap.Error(err), zap.String("url", url))
		os.RemoveAll(uniqueDir)
		return "", nil, fmt.Errorf("failed to download video: %w", err)
	}

	files, err := os.ReadDir(uniqueDir)
	if err != nil {
		os.RemoveAll(uniqueDir)
		return "", nil, fmt.Errorf("failed to read download directory: %w", err)
	}

	var videoFile, infoFile string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			infoFile = filepath.Join(uniqueDir, file.Name())
		} else if !file.IsDir() {
			videoFile = filepath.Join(uniqueDir, file.Name())
		}
	}

	if videoFile == "" {
		os.RemoveAll(uniqueDir)
		return "", nil, fmt.Errorf("video file not found after download")
	}

	var info *VideoInfo
	if infoFile != "" {
		infoData, err := os.ReadFile(infoFile)
		if err != nil {
			s.logger.Warn("Failed to read info file", zap.Error(err))
		} else {
			info = &VideoInfo{}
			if err := json.Unmarshal(infoData, info); err != nil {
				s.logger.Warn("Failed to parse info file", zap.Error(err))
				info = nil
			}
		}
	}

	s.logger.Info("Video downloaded successfully", 
		zap.String("file", videoFile), 
		zap.String("url", url))

	return videoFile, info, nil
}

func (s *Service) GetVideoReader(filePath string) (io.ReadCloser, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open video file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to get file stats: %w", err)
	}

	return file, stat.Size(), nil
}

func (s *Service) CleanupFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.RemoveAll(dir); err != nil {
		s.logger.Error("Failed to cleanup download directory", zap.Error(err), zap.String("path", dir))
		return err
	}
	s.logger.Debug("Cleaned up download directory", zap.String("path", dir))
	return nil
}