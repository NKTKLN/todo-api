package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary   Create subtask
// @Tags      Working with subtasks
// @Accept    json
// @Produce   json
// @Param     SubtaskData  body      models.ApiSubtaskData  true  "Subtask data"
// @Success   200          {object}  models.ApiMessage
// @Failure   400          {object}  models.ApiError
// @Failure   404          {object}  models.ApiError
// @Failure   500          {object}  models.ApiError
// @Security  token
// @Router    /todo/subtask/add [post]
func (h *Handler) AddSubtask(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "comment": "Sugar-free",
		  "name": "Coca-Cola",
		  "task_id": 1023456789
		}
	*/

	var data models.ApiSubtaskData
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.GetListIdWhereTask(userId, data.TaskId) == 0:
		NewErrorResponse(c, http.StatusNotFound, "This task not found.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	}
	if c.IsAborted() {
		return
	}

	// Create new subtask
	err := h.PostgresDB.CreateSubtask(models.Tasks{TaskId: data.TaskId, Name: data.Name, Comment: data.Comment})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Subtask added to db.",
	})
}

// @Summary   Delete subtask
// @Tags      Working with subtasks
// @Accept    json
// @Produce   json
// @Param     subtask_id  query     int  true  "The id of the subtask to be deleted"
// @Success   200         {object}  models.ApiMessage
// @Failure   404         {object}  models.ApiError
// @Failure   500         {object}  models.ApiError
// @Security  token
// @Router    /todo/subtask/delete [delete]
func (h *Handler) DeleteSubtask(c *gin.Context) {
	subtaskId, err := strconv.Atoi(c.Query("subtask_id"))
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	taskId := h.PostgresDB.GetTaskIdWhereSubtask(subtaskId)
	listId := h.PostgresDB.GetListIdWhereTask(userId, taskId)

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting subtask_id.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case listId == 0:
		NewErrorResponse(c, http.StatusNotFound, "This subtask not found.")
	}
	if c.IsAborted() {
		return
	}

	// Update index
	subtaskIndex := h.PostgresDB.GetTaskById(subtaskId).Index
	for _, editTask := range h.PostgresDB.GetSubtasksForEditIndex(taskId, subtaskIndex) {
		err := h.PostgresDB.UpdateTaskIndex(editTask.Id, editTask.Index-1)
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Delete subtask
	if err := h.PostgresDB.DeleteSubtask(subtaskId); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "The subtask has been deleted.",
	})
}

// @Summary   Edit subtask
// @Tags      Working with subtasks
// @Accept    json
// @Produce   json
// @Param     SubtaskData  body      models.SubtaskEditData  true  "Subtask data"
// @Success   200          {object}  models.ApiMessage
// @Failure   400          {object}  models.ApiError
// @Failure   404          {object}  models.ApiError
// @Failure   500          {object}  models.ApiError
// @Security  token
// @Router    /todo/subtask/edit [put]
func (h *Handler) EditSubtask(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "categories": [
		  	"Party",
		  	"Shoping"
		  ],
		  "comment": "Sugar-free",
		  "done": true,
		  "end_time": "2077-12-10 13:13",
		  "id": 1023456789,
		  "index": 0,
		  "name": "Coca-Cola",
		  "special": true
		}
	*/

	var data models.SubtasksData
	if c.ShouldBindJSON(&data) != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
		return
	}
	endTime, err := time.Parse("2006-01-02 15:04", data.EndTime)
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	taskId := h.PostgresDB.GetTaskIdWhereSubtask(data.Id)
	listId := h.PostgresDB.GetListIdWhereTask(userId, taskId)

	// Input data check
	switch {
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case listId == 0:
		NewErrorResponse(c, http.StatusNotFound, "This subtask not found.")
	case err != nil:
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect time format.")
	case endTime.Before(time.Now()):
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect time.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	case data.Index < 0 || data.Index > h.PostgresDB.GetSubtaskMaxIndex(taskId):
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect index.")
	}
	if c.IsAborted() {
		return
	}

	// Updating subtask data
	err = h.PostgresDB.UpdateTaskData(models.Tasks{Id: data.Id, Name: data.Name, Comment: data.Comment, Categories: data.Categories, EndTime: endTime, Done: data.Done, Special: data.Special})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Updating subtask index
	taskIndex := h.PostgresDB.GetTaskById(data.Id).Index
	if taskIndex != data.Index {
		err := h.PostgresDB.UpdateSubtasksIndexes(models.Tasks{Id: data.Id, TaskId: taskId, Index: data.Index})
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Updating the task data was successful.",
	})
}

// @Summary   Shows all subtasks in the task
// @Tags      Working with subtasks
// @Accept    json
// @Produce   json
// @Param     task_id  query     int  true  "Task id with subtasks"
// @Success   200      {object}  models.ApiShowSubtasks
// @Failure   404      {object}  models.ApiError
// @Failure   500      {object}  models.ApiError
// @Security  token
// @Router    /todo/subtask/show [get]
func (h *Handler) ShowSubtasks(c *gin.Context) {
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	taskId, err := strconv.Atoi(c.Query("task_id"))

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting task_id.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.GetListIdWhereTask(userId, taskId) == 0:
		NewErrorResponse(c, http.StatusNotFound, "This list not found.")
	}
	if c.IsAborted() {
		return
	}

	c.JSON(http.StatusOK, models.ApiShowSubtasks{
		Subtasks: h.PostgresDB.GetAllSubtasks(taskId),
	})
}
