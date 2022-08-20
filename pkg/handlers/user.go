package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary  Get user icon
// @Tags     Show user data
// @Accept   json
// @Produce  json
// @Param    user_id  query     int  true  "User id"
// @Failure  400      {object}  models.ApiError
// @Failure  404      {object}  models.ApiError
// @Failure  500      {object}  models.ApiError
// @Router   /user/show/icon [get]
func (h *Handler) GetUserIcon(c *gin.Context) {
	userId, err := strconv.Atoi(c.Query("user_id"))
	if err != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting user_id.")
		return
	}
	userData := h.PostgresDB.GetUserById(userId)

	// Input data check
	switch {
	case userData.Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "User not found.")
	case userData.Icon == "":
		NewErrorResponse(c, http.StatusNotFound, "The user icon is not yet installed.")
	}
	if c.IsAborted() {
		return
	}

	reader, err := h.MinIOClient.DownloadFile(c.Request.Context(), userData.Icon)
	if err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Problem with retrieving an image from the database.")
		return
	}
	defer reader.Close()

	info, err := reader.Stat()
	if err != nil {
		NewErrorResponse(c, http.StatusBadRequest, err.Error())
		return 
	}

	iconType, ex := models.IMAGE_TYPES[info.ContentType]
	if !ex {
		NewErrorResponse(c, http.StatusBadRequest, "The problem with extracting an image type.")
		return 
	}

	extraHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="icon%s"`, iconType),
	}

	c.DataFromReader(http.StatusOK, info.Size, info.ContentType, reader, extraHeaders)
}

// @Summary  Get basic user data
// @Tags     Show user data
// @Accept   json
// @Produce  json
// @Param    user_id  query     int  true  "User id"
// @Success  200      {object}  models.ShowUserData
// @Failure  404      {object}  models.ApiError
// @Failure  500      {object}  models.ApiError
// @Router   /user/show/data-by-id [get]
func (h *Handler) GetUserData(c *gin.Context) {
	userId, err := strconv.Atoi(c.Query("user_id"))
	if err != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Error when converting user_id.")
		return
	}
	userData := h.PostgresDB.GetUserById(userId)

	// Input data check
	if userData.Id == 0 {
		NewErrorResponse(c, http.StatusNotFound, "User not found.")
		return
	}

	// Generating user output data
	var outputData models.ShowUserData
	if err := copier.Copy(&outputData, &userData); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, outputData)
}

// @Summary  Get basic user data
// @Tags     Show user data
// @Accept   json
// @Produce  json
// @Success  200  {object}  models.ShowUserData
// @Failure  404  {object}  models.ApiError
// @Failure  500  {object}  models.ApiError
// @Security token
// @Router   /user/show/data-by-token [get]
func (h *Handler) GetUserDataByToken(c *gin.Context) {
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	userData := h.PostgresDB.GetUserById(userId)

	// Input data check
	if userData.Id == 0 {
		NewErrorResponse(c, http.StatusNotFound, "User not found.")
		return
	}

	// Generating user output data
	var outputData models.ShowUserData
	if err := copier.Copy(&outputData, &userData); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, outputData)
}
