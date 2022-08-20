package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/NKTKLN/todo-api/pkg/db"
)

type MinioProvider struct {
	minioAuthData
	client *minio.Client
}

type minioAuthData struct {
	url      string
	user     string
	password string
	ssl      bool
}

// Creating new provider for MinIO
func NewMinioProvider(minioURL, minioUser, minioPassword string, ssl bool) db.MinIOClient {
	return &MinioProvider{
		minioAuthData: minioAuthData{
			password: minioPassword,
			url:      minioURL,
			user:     minioUser,
			ssl:      ssl,
		},
	}
}

// Connecting to a MinIO database
func (m *MinioProvider) Connect() (err error) {
	m.client, err = minio.New(m.url, &minio.Options{
		Creds:  credentials.NewStaticV4(m.user, m.password, ""),
		Secure: m.ssl,
	})

	return err
}
