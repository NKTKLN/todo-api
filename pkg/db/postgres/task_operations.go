package postgres

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/NKTKLN/todo-api/models"
)

func (d *PDB) CreateTask(model models.Tasks) error {
	// Generating new data for the task
	taskId := int(uuid.New().ID())
	for !d.checkTaskId(taskId) {
		taskId = int(uuid.New().ID())
	}

	var index int
	if len(d.GetAllTasks(model.ListId)) > 0 {
		index = d.GetTaskMaxIndex(model.ListId) + 1
	}

	// Creating new task
	return d.DB.Table("tasks").Create(&models.Tasks{Id: taskId, ListId: model.ListId, Name: model.Name, Comment: model.Comment, Index: index}).Error
}

func (d *PDB) checkTaskId(id int) bool {
	var taskData models.Tasks
	result := d.DB.Table("tasks").Where("id = ?", id).Take(&taskData).Error
	return errors.Is(result, gorm.ErrRecordNotFound)
}

func (d *PDB) GetAllTasks(listId int) (tasksData []models.TasksData) {
	var tasks []models.Tasks
	d.DB.Table("tasks").Where("list_id = ?", listId).Order("index").Find(&tasks)

	if copier.Copy(&tasksData, &tasks) != nil {
		return
	}

	for index, task := range tasks {
		tasksData[index].EndTime = task.EndTime.Format("2006-01-02 15:04")
	}
	return
}

func (d *PDB) GetTaskById(id int) (taksData models.Tasks) {
	d.DB.Table("tasks").Where("id = ?", id).Take(&taksData)
	return
}

func (d *PDB) GetTasksForEditIndex(listId, taskIndex int) (taskData []models.Tasks) {
	d.DB.Table("tasks").Where("list_id = ? AND index > ?", listId, taskIndex).Find(&taskData)
	return
}

func (d *PDB) GetListIdWhereTask(userId, taskId int) (listId int) {
	d.DB.Table("lists").Select("lists.id").Joins("INNER JOIN tasks ON lists.id=tasks.list_id").Where("user_id = ? AND tasks.id = ?", userId, taskId).Take(&listId)
	return
}

func (d *PDB) GetTaskMaxIndex(listId int) (index int) {
	d.DB.Table("tasks").Select("max(index)").Where("list_id = ?", listId).Take(&index)
	return
}

func (d *PDB) UpdateTaskData(model models.Tasks) error {
	return d.DB.Table("tasks").Select("name", "comment", "categories", "end_time", "done", "special").Updates(model).Error
}

func (d *PDB) UpdateTaskIndex(id, index int) error {
	return d.DB.Table("tasks").Where("id = ?", id).Update("index", index).Error
}

func (d *PDB) UpdateTasksIndexes(model models.Tasks) (err error) {
	step := 1
	taskIndex := d.GetTaskById(model.Id).Index

	// Obtaining tasks for the update
	var listsTasks []models.Tasks
	if taskIndex > model.Index {
		d.DB.Table("tasks").Where("list_id = ? AND index >= ? AND index < ?", model.ListId, model.Index, taskIndex).Find(&listsTasks)
	} else {
		d.DB.Table("tasks").Where("list_id = ? AND index <= ? AND index > ?", model.ListId, model.Index, taskIndex).Find(&listsTasks)
		step *= -1
	}

	// Updating tasks indexes
	for _, task := range listsTasks {
		err = d.UpdateTaskIndex(task.Id, task.Index+step)
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

func (d *PDB) DeleteTask(id int) error {
	// Deleting all task subtasks
	for _, subtask := range d.GetAllSubtasks(id) {
		if err := d.DeleteSubtask(subtask.Id); err != nil {
			return err
		}
	}

	// Deleting task
	return d.DB.Table("tasks").Delete(&models.Tasks{}, id).Error
}
