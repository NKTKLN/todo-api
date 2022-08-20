package models

import (
	"time"

	"github.com/lib/pq"
)

type Users struct {
	Id       int
	Email    string
	Password string
	Name     string
	Username string
	Icon     string
}

type Lists struct {
	Id      int
	UserId  int
	Name    string
	Comment string
	Index   int
}

type Tasks struct {
	Id         int
	ListId     int
	TaskId     int
	Name       string
	Comment    string
	Index      int
	Categories pq.StringArray `gorm:"type:text[]"`
	EndTime    time.Time
	Done       bool
	Special    bool
}
