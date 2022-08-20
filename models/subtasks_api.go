package models

import "github.com/lib/pq"

type ApiShowSubtasks struct {
	Subtasks []SubtasksData `json:"subtasks"`
}

type ApiSubtaskData struct {
	TaskId  int    `json:"task_id" example:"1023456789"`
	Name    string `json:"name" example:"Coca-Cola"`
	Comment string `json:"comment" example:"Sugar-free"`
}

type SubtasksData struct {
	Id         int            `json:"id" example:"1023456789"`
	Name       string         `json:"name" example:"Coca-Cola"`
	Comment    string         `json:"comment" example:"Sugar-free"`
	Index      int            `json:"index" example:"0"`
	Categories pq.StringArray `gorm:"type:text[]" json:"categories" example:"Party,Shoping"`
	EndTime    string         `json:"end_time" example:"2077-12-10 13:13"`
	Done       bool           `json:"done"`
	Special    bool           `json:"special"`
}

type SubtaskEditData struct {
	Id         int            `json:"id" example:"1023456789"`
	Name       string         `json:"name" example:"Pepsi"`
	Comment    string         `json:"comment" example:"Sugar-free"`
	Index      int            `json:"index" example:"1"`
	Categories pq.StringArray `gorm:"type:text[]" json:"categories" example:"Party,Shoping,Today"`
	EndTime    string         `json:"end_time" example:"2077-12-10 13:13"`
	Done       bool           `json:"done" example:"true"`
	Special    bool           `json:"special" example:"true"`
}
