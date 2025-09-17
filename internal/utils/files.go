package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func GetContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".m4v":
		return "video/x-m4v"
	case ".3gp":
		return "video/3gpp"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".opus":
		return "audio/opus"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func GenerateUniqueKey(originalFilename string) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	
	ext := filepath.Ext(originalFilename)
	baseName := strings.TrimSuffix(originalFilename, ext)
	safeName := SanitizeFilename(baseName)
	
	return fmt.Sprintf("%d_%s_%s%s", timestamp, randomStr, safeName, ext)
}

func SanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	sanitized := reg.ReplaceAllString(filename, "_")
	
	sanitized = strings.Trim(sanitized, "_")
	
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	if sanitized == "" {
		sanitized = "file"
	}
	
	return sanitized
}