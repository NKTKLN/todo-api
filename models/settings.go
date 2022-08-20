package models

import "time"

const (
	MAX_ICON_UPLOAD_SIZE = 5 << 20             // 5MB
	ACCESS_TOKEN_LIVE    = 15 * time.Minute    // 15 minutes
	REFRESH_TOKEN_LIVE   = 30 * 24 * time.Hour // 30 days
)

var (
	IMAGE_TYPES = map[string]interface{}{
		"image/jpeg": ".jpeg",
		"image/png":  ".png",
	}
)

const BucketName = "user-icons"
