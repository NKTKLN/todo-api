package postgres

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/NKTKLN/todo-api/models"
)

func (d *PDB) CreateList(model models.Lists) error {
	// Generating new data for the list
	listId := int(uuid.New().ID())
	for !d.checkListId(listId) {
		listId = int(uuid.New().ID())
	}

	var index int
	if len(d.GetAllUserLists(model.UserId)) > 0 {
		index = d.GetListMaxIndex(model.UserId) + 1
	}

	// Creating new list
	return d.DB.Table("lists").Create(&models.Lists{Id: listId, UserId: model.UserId, Name: model.Name, Comment: model.Comment, Index: index}).Error
}

func (d *PDB) checkListId(id int) bool {
	var listData models.Lists
	result := d.DB.Table("lists").Where("id = ?", id).Take(&listData).Error
	return errors.Is(result, gorm.ErrRecordNotFound)
}

func (d *PDB) GetAllUserLists(userId int) (listsData []models.ListsData) {
	d.DB.Table("lists").Where("user_id = ?", userId).Order("index").Find(&listsData)
	if len(listsData) == 0 {
		return nil
	}
	return
}

func (d *PDB) GetListsForEditIndex(userId, listIndex int) (listData []models.Lists) {
	d.DB.Table("lists").Where("user_id = ? AND index > ?", userId, listIndex).Find(&listData)
	return
}

func (d *PDB) GetListById(id int) (listData models.Lists) {
	d.DB.Table("lists").Where("id = ?", id).Take(&listData)
	return
}

func (d *PDB) GetListByIdAndUserId(id, userId int) (listData models.Lists) {
	d.DB.Table("lists").Where("id = ? AND user_id = ?", id, userId).Take(&listData)
	return
}

func (d *PDB) GetListMaxIndex(userId int) (index int) {
	d.DB.Table("lists").Select("max(index)").Where("user_id = ?", userId).Take(&index)
	return
}

func (d *PDB) UpdateListData(model models.Lists) error {
	return d.DB.Table("lists").Select("name", "comment").Updates(model).Error
}

func (d *PDB) UpdateListIndex(listId, index int) error {
	return d.DB.Table("lists").Where("id = ?", listId).Update("index", index).Error
}

func (d *PDB) UpdateListsIndexes(model models.Lists) (err error) {
	step := 1
	listIndex := d.GetListById(model.Id).Index

	// Obtaining lists for the update
	var userLists []models.Lists
	if listIndex > model.Index {
		d.DB.Table("lists").Where("user_id = ? AND index >= ? AND index < ?", model.UserId, model.Index, listIndex).Find(&userLists)
	} else {
		d.DB.Table("lists").Where("user_id = ? AND index <= ? AND index > ?", model.UserId, model.Index, listIndex).Find(&userLists)
		step *= -1
	}

	// Updating lists indexes
	for _, userList := range userLists {
		if err = d.UpdateListIndex(userList.Id, userList.Index+step); err != nil {
			return
		}
	}
	err = d.UpdateListIndex(model.Id, model.Index)
	return
}

func (d *PDB) DeleteList(id int) error {
	// Deleting all list tasks
	for _, task := range d.GetAllTasks(id) {
		if err := d.DeleteTask(task.Id); err != nil {
			return err
		}
	}

	// Deleting list
	return d.DB.Delete(&models.Lists{}, id).Error
}
