package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary   Create list
// @Tags      Working with lists
// @Accept    json
// @Produce   json
// @Param     ListData  body      models.ApiListData  true  "List data"
// @Success   200       {object}  models.ApiMessage
// @Failure   400       {object}  models.ApiError
// @Failure   404       {object}  models.ApiError
// @Failure   500       {object}  models.ApiError
// @Security  token
// @Router    /todo/list/add [post]
func (h *Handler) AddList(c *gin.Context) {
	/*
		Example of JSON received

		{
			"comment": "Products needed for the party",
			"name": "List of products"
		}
	*/

	var data models.ApiListData
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	}
	if c.IsAborted() {
		return
	}

	// Create new list
	err := h.PostgresDB.CreateList(models.Lists{UserId: userId, Name: data.Name, Comment: data.Comment})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "List added to db.",
	})
}

// @Summary   Delete list
// @Tags      Working with lists
// @Accept    json
// @Produce   json
// @Param     list_id  query     int  true  "The id of the list to be deleted"
// @Success   200      {object}  models.ApiMessage
// @Failure   404      {object}  models.ApiError
// @Failure   500      {object}  models.ApiError
// @Security  token
// @Router   /todo/list/delete [delete]
func (h *Handler) DeleteList(c *gin.Context) {
	listId, err := strconv.Atoi(c.Query("list_id"))
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	listData := h.PostgresDB.GetListByIdAndUserId(listId, userId)

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting list_id.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case listData.Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "This list not found.")
	}
	if c.IsAborted() {
		return
	}

	// Updating list index
	for _, editList := range h.PostgresDB.GetListsForEditIndex(userId, listData.Index) {
		err := h.PostgresDB.UpdateListIndex(editList.Id, editList.Index-1)
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	
	// Delete list
	if err := h.PostgresDB.DeleteList(listId); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "The list has been deleted.",
	})
}

// @Summary   Edit list
// @Tags      Working with lists
// @Accept    json
// @Produce   json
// @Param     ListData  body      models.ListEditData  true  "List data"
// @Success   200       {object}  models.ApiMessage
// @Failure   400       {object}  models.ApiError
// @Failure   404       {object}  models.ApiError
// @Failure   500       {object}  models.ApiError
// @Security  token
// @Router    /todo/list/edit [put]
func (h *Handler) EditList(c *gin.Context) {
	/*
		Example of JSON received

		{
			"comment": "New products needed for the party",
			"id": 1023456789,
			"index": 1,
			"name": "New list of products"
		}
	*/

	var data models.ListsData
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.GetListByIdAndUserId(data.Id, userId).Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "This list not found.")
	case data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Empty name.")
	case len(data.Name) > 32: 
		NewErrorResponse(c, http.StatusBadRequest, "A name longer than 32 characters.")
	case data.Index < 0 || data.Index > h.PostgresDB.GetListMaxIndex(userId):
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect index.")
	}
	if c.IsAborted() {
		return
	}

	// Updating list data
	err := h.PostgresDB.UpdateListData(models.Lists{Id: data.Id, Name: data.Name, Comment: data.Comment})
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Updating list index
	if h.PostgresDB.GetListById(data.Id).Index != data.Index {
		err := h.PostgresDB.UpdateListsIndexes(models.Lists{Id: data.Id, UserId: userId, Index: data.Index})
		if err != nil {
			NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Updating the list data was successful.",
	})
}

// @Summary   Shows all lists created by the user
// @Tags      Working with lists
// @Accept    json
// @Produce   json
// @Success   200  {object}  models.ApiShowLists
// @Failure   404  {object}  models.ApiError
// @Security  token
// @Router    /todo/list/show [get]
func (h *Handler) ShowLists(c *gin.Context) {
	// Checking a token in the db
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	if userId == 0 {
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
		return
	}
	
	// Get data from the db
	c.JSON(http.StatusOK, models.ApiShowLists{
		Lists: h.PostgresDB.GetAllUserLists(userId),
	})
}
