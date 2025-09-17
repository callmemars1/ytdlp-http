# YT-DLP HTTP Server

HTTP server for downloading videos using yt-dlp with direct streaming or S3-compatible storage upload.

## Quick Start

```bash
docker run -p 8080:8080 \
  -e S3_ACCESS_KEY_ID=your_key \
  -e S3_SECRET_ACCESS_KEY=your_secret \
  -e S3_REGION=us-east-1 \
  -e S3_BUCKET=your_bucket \
  -e S3_ENDPOINT=http://your-minio:9000 \
  ghcr.io/callmemars1/ytdlp-http:latest
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SERVER_ADDR` | No | `:8080` | Server listen address |
| `S3_ACCESS_KEY_ID` | Yes | - | S3 access key |
| `S3_SECRET_ACCESS_KEY` | Yes | - | S3 secret key |
| `S3_REGION` | Yes | - | S3 region |
| `S3_BUCKET` | Yes | - | S3 bucket name |
| `S3_ENDPOINT` | Yes | - | S3 endpoint URL |
| `AUTH_ENABLED` | No | `false` | Enable authentication |
| `AUTH_API_KEY` | No | - | SHA256 hash of API key |

## API Endpoints

### POST /download

Downloads video and streams it directly as response.

**Request:**
```json
{
  "url": "https://www.youtube.com/watch?v=example",
  "options": {
    "format": "best[height<=720]",
    "quality": "720"
  },
  "timeout": 300
}
```

**Response:** Binary video stream with appropriate headers.

### POST /upload

Downloads video and uploads to S3 with metadata JSON file.

**Request:**
```json
{
  "url": "https://www.youtube.com/watch?v=example",
  "s3_key": "my-video",
  "options": {
    "format": "best[height<=720]"
  },
  "timeout": 600
}
```

**Response:**
```json
{
  "success": true,
  "message": "Video uploaded successfully to S3-compatible storage",
  "result": {
    "video_upload": {
      "key": "1234567890_abcd1234_my-video.mp4",
      "bucket": "ytdlp-downloads",
      "location": "http://minio:9000/ytdlp-downloads/1234567890_abcd1234_my-video.mp4",
      "size": 15728640
    },
    "metadata_upload": {
      "key": "1234567890_abcd1234_my-video.json",
      "bucket": "ytdlp-downloads",
      "location": "http://minio:9000/ytdlp-downloads/1234567890_abcd1234_my-video.json",
      "size": 2048
    },
    "total_size": 15730688,
    "uploaded_at": "2024-01-01T12:00:00Z"
  }
}
```

## Authentication

When `AUTH_ENABLED=true`, include Bearer token in Authorization header:

```bash
curl -H "Authorization: Bearer your-api-key" \
     -H "Content-Type: application/json" \
     -d '{"url":"https://example.com/video"}' \
     http://localhost:8080/download
```

Generate API key hash:
```bash
echo -n "your-secret-api-key" | sha256sum
```