package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"

	"github.com/NKTKLN/todo-api/models"
)

func (m *MinioProvider) CreateBucket(ctx context.Context) error {
	err := m.client.MakeBucket(ctx, models.BucketName, minio.MakeBucketOptions{})
    if err == nil {
		return nil
    }
	
	exist, errBucketExists := m.client.BucketExists(ctx, models.BucketName)
	switch {
	case errBucketExists != nil:
		return errBucketExists
	case !exist:
		return err
	default:
		return nil
	}
}

func (m *MinioProvider) UploadFile(ctx context.Context, input models.FileUnit) (imageName string, err error) {
	if err = m.CreateBucket(ctx); err != nil {
		return
	}

	imageName = fmt.Sprintf("user-%d%s", input.ID, models.IMAGE_TYPES[input.ContentType])

	_, err = m.client.PutObject(
		ctx,
		models.BucketName,
		imageName,
		input.Icon,
		input.Size,
		minio.PutObjectOptions{ContentType: input.ContentType},
	)

	return
}

func (m *MinioProvider) DownloadFile(ctx context.Context, filename string) (*minio.Object, error) {
	if err := m.CreateBucket(ctx); err != nil {
		return nil, err
	}

	return m.client.GetObject(
		ctx,
		models.BucketName,
		filename,
		minio.GetObjectOptions{},
	)
}

func (m *MinioProvider) DeleteFile(ctx context.Context, filename string) error {
	if err := m.CreateBucket(ctx); err != nil {
		return err
	}

	return m.client.RemoveObject(
		ctx,
		models.BucketName,
		filename,
		minio.RemoveObjectOptions{ForceDelete: true},
	)
}
