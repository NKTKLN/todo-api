package models

import "github.com/lib/pq"

type ApiShowTasks struct {
	Tasks []TasksData `json:"tasks"`
}

type ApiTaskData struct {
	ListId  int    `json:"list_id" example:"1023456789"`
	Name    string `json:"name" example:"Buy drinks"`
	Comment string `json:"comment" example:"Go to the supermarket on the way home"`
}

type TasksData struct {
	Id         int            `json:"id" example:"1023456789"`
	Name       string         `json:"name" example:"Buy drinks"`
	Comment    string         `json:"comment" example:"Go to the supermarket on the way home"`
	Index      int            `json:"index" example:"0"`
	Categories pq.StringArray `gorm:"type:text[]" json:"categories" example:"Party,Shoping"`
	EndTime    string         `json:"end_time" example:"2077-12-10 13:13"`
	Done       bool           `json:"done"`
	Special    bool           `json:"special"`
}

type TaskEditData struct {
	Id         int            `json:"id" example:"1023456789"`
	Name       string         `json:"name" example:"Buy new drinks"`
	Comment    string         `json:"comment" example:"Go to the supermarket on the way home"`
	Index      int            `json:"index" example:"1"`
	Categories pq.StringArray `gorm:"type:text[]" json:"categories" example:"Party,Shoping,Today"`
	EndTime    string         `json:"end_time" example:"2077-12-10 13:13"`
	Done       bool           `json:"done" example:"true"`
	Special    bool           `json:"special" example:"true"`
}
