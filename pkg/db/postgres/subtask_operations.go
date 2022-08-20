package postgres

import (
	"github.com/google/uuid"
	"github.com/jinzhu/copier"

	"github.com/NKTKLN/todo-api/models"
)

func (d *PDB) CreateSubtask(model models.Tasks) error {
	// Generating new data for the subtask
	subtaskId := int(uuid.New().ID())
	for !d.checkTaskId(subtaskId) {
		subtaskId = int(uuid.New().ID())
	}
	var index int
	if len(d.GetAllSubtasks(model.TaskId)) > 0 {
		index = d.GetSubtaskMaxIndex(model.TaskId) + 1
	}

	// Creating new subtask
	return d.DB.Table("tasks").Create(&models.Tasks{Id: subtaskId, TaskId: model.TaskId, Name: model.Name, Comment: model.Comment, Index: index}).Error
}

func (d *PDB) GetAllSubtasks(taskId int) (subTasksData []models.SubtasksData) {
	var subtasks []models.Tasks
	d.DB.Table("tasks").Where("task_id = ?", taskId).Order("index").Find(&subtasks)
	
	if copier.Copy(&subTasksData, &subtasks) != nil {
		return
	}

	for index, task := range subtasks {
		subTasksData[index].EndTime = task.EndTime.Format("2006-01-02 15:04")
	}
	return
}

func (d *PDB) GetSubtasksForEditIndex(taskId, subtaskIndex int) (subtaskData []models.Tasks) {
	d.DB.Table("tasks").Where("task_id = ? AND index > ?", taskId, subtaskIndex).Find(&subtaskData)
	return
}

func (d *PDB) GetTaskIdWhereSubtask(subtaskId int) (taskId int) {
	d.DB.Table("tasks").Select("task_id").Where("id = ?", subtaskId).Take(&taskId)
	return
}

func (d *PDB) GetSubtaskMaxIndex(taskId int) (index int) {
	d.DB.Table("tasks").Select("max(index)").Where("task_id = ?", taskId).Take(&index)
	return
}

func (d *PDB) UpdateSubtasksIndexes(model models.Tasks) (err error) {
	step := 1
	subtaskIndex := d.GetTaskById(model.Id).Index

	// Obtaining subtasks for the update
	var taskSubtasks []models.Tasks
	if subtaskIndex > model.Index {
		d.DB.Table("tasks").Where("task_id = ? AND index >= ? AND index < ?", model.TaskId, model.Index, subtaskIndex).Find(&taskSubtasks)
	} else {
		d.DB.Table("tasks").Where("task_id = ? AND index <= ? AND index > ?", model.TaskId, model.Index, subtaskIndex).Find(&taskSubtasks)
		step *= -1
	}

	// Updating subtasks indexes
	for _, subtask := range taskSubtasks {
		err = d.UpdateTaskIndex(subtask.Id, subtask.Index+step)
		if err != nil {
			return
		}
	}
	err = d.UpdateTaskIndex(model.Id, model.Index)
	if err != nil {
		return
	}
	return
}

func (d *PDB) DeleteSubtask(id int) error {
	return d.DB.Table("tasks").Delete(&models.Tasks{}, id).Error
}
