package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary   Create task
// @Tags      Working with tasks
// @Accept    json
// @Produce   json
// @Param     TaskData  body      models.ApiTaskData  true  "Task data"
// @Success   200       {object}  models.ApiMessage
// @Failure   400       {object}  models.ApiError
// @Failure   404       {object}  models.ApiError
// @Failure   500       {object}  models.ApiError
// @Security  token
// @Router    /todo/task/add [post]
func (h *Handler) AddTask(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "comment": "Go to the supermarket on the way home",
		  "list_id": 1023456789,
		  "name": "Buy drinks"
		}
	*/

	var data models.ApiTaskData
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.GetListByIdAndUserId(data.ListId, userId).Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "This list not found.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	}
	if c.IsAborted() {
		return
	}

	// Create new task
	err := h.PostgresDB.CreateTask(models.Tasks{ListId: data.ListId, Name: data.Name, Comment: data.Comment})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Task added to db.",
	})
}

// @Summary   Delete task
// @Tags      Working with tasks
// @Accept    json
// @Produce   json
// @Param     task_id  query     int  true  "The id of the task to be deleted"
// @Success   200      {object}  models.ApiMessage
// @Failure   404      {object}  models.ApiError
// @Failure   500      {object}  models.ApiError
// @Security  token
// @Router    /todo/task/delete [delete]
func (h *Handler) DeleteTask(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Query("task_id"))
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	listId := h.PostgresDB.GetListIdWhereTask(userId, taskId)

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting task_id.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case listId == 0:
		NewErrorResponse(c, http.StatusNotFound, "This task not found.")
	}
	if c.IsAborted() {
		return
	}

	// Update index
	taskIndex := h.PostgresDB.GetTaskById(taskId).Index
	for _, editTask := range h.PostgresDB.GetTasksForEditIndex(listId, taskIndex) {
		err := h.PostgresDB.UpdateTaskIndex(editTask.Id, editTask.Index-1)
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Delete task
	if err := h.PostgresDB.DeleteTask(taskId); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "The task has been deleted.",
	})
}

// @Summary   Edit task
// @Tags      Working with tasks
// @Accept    json
// @Produce   json
// @Param     TaskData  body      models.TaskEditData  true  "Task data"
// @Success   200       {object}  models.ApiMessage
// @Failure   400       {object}  models.ApiError
// @Failure   404       {object}  models.ApiError
// @Failure   500       {object}  models.ApiError
// @Security  token
// @Router    /todo/task/edit [put]
func (h *Handler) EditTask(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "categories": [
		  	"Party",
		  	"Shoping"
		  ],
		  "comment": "Go to the supermarket on the way home",
		  "done": true,
		  "end_time": "2077-12-10 13:13",
		  "id": 1023456789,
		  "index": 0,
		  "name": "Buy drinks",
		  "special": true
		}
	*/

	var data models.TasksData
	if c.ShouldBindJSON(&data) != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
		return
	}
	endTime, err := time.Parse("2006-01-02 15:04", data.EndTime)
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	listId := h.PostgresDB.GetListIdWhereTask(userId, data.Id)

	// Input data check
	switch {
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case listId == 0:
		NewErrorResponse(c, http.StatusNotFound, "This task not found.")
	case err != nil:
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect time format.")
	case endTime.Before(time.Now()):
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect time.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	case data.Index < 0 || data.Index > h.PostgresDB.GetTaskMaxIndex(listId):
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect index.")
	}
	if c.IsAborted() {
		return
	}

	// Updating task data
	err = h.PostgresDB.UpdateTaskData(models.Tasks{Id: data.Id, Name: data.Name, Comment: data.Comment, Categories: data.Categories, EndTime: endTime, Done: data.Done, Special: data.Special})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Updating task index
	taskIndex := h.PostgresDB.GetTaskById(data.Id).Index
	if taskIndex != data.Index {
		err := h.PostgresDB.UpdateTasksIndexes(models.Tasks{Id: data.Id, ListId: listId, Index: data.Index})
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Updating the task data was successful.",
	})
}

// @Summary   Shows all tasks in the list
// @Tags      Working with tasks
// @Accept    json
// @Produce   json
// @Param     list_id  query     int  true  "List id with tasks"
// @Success   200      {object}  models.ApiShowTasks
// @Failure   404      {object}  models.ApiError
// @Failure   500      {object}  models.ApiError
// @Security  token
// @Router    /todo/task/show [get]
func (h *Handler) ShowTasks(c *gin.Context) {
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	listId, err := strconv.Atoi(c.Query("list_id"))

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting list_id.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.GetListByIdAndUserId(listId, userId).Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "This list not found.")
	}
	if c.IsAborted() {
		return
	}

	c.JSON(http.StatusOK, models.ApiShowTasks{
		Tasks: h.PostgresDB.GetAllTasks(listId),
	})
}
