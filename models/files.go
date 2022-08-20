package models

import "io"

type FileUnit struct {
	Icon        io.Reader
	Size        int64
	ContentType string
	ID          int
}
