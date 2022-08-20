package handlers

import (
	"bytes"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary   Change user name
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Param     NewUserName  body      models.UserName  true  "User name"
// @Success   200          {object}  models.ApiMessage
// @Failure   404          {object}  models.ApiError
// @Failure   500          {object}  models.ApiError
// @Security  token
// @Router    /user/settings/update/name [patch]
func (h *Handler) EditUserName(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "name": "NKTKLN"
		}
	*/

	var data models.UserName
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	}
	if c.IsAborted() {
		return
	}

	nameMatched, err := regexp.MatchString(`^[a-zA-Z]+([ ]?[a-zA-Z0-9]+){0,2}$`, data.Name)
	if !nameMatched || err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect name.")
		return
	}

	// Updating a user's name
	if err := h.PostgresDB.UpdateUser(models.Users{Id: userId}, models.Users{Name: data.Name}); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Name updated successfully.",
	})
}

// @Summary   Change user username
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Param     NewUsername  body      models.UserUsername  true  "Username"
// @Success   200          {object}  models.ApiMessage
// @Failure   404          {object}  models.ApiError
// @Failure   500          {object}  models.ApiError
// @Security  token
// @Router    /user/settings/update/username [patch]
func (h *Handler) EditUserUsername(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "username": "NKTKLN"
		}
	*/

	var data models.UserUsername
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case !h.PostgresDB.CheckUserUsername(data.Username):
		NewErrorResponse(c, http.StatusBadRequest, "This username is already in use.")
	}
	if c.IsAborted() {
		return
	}

	usernameMatched, err := regexp.MatchString(`^[a-z]+([-_]?[a-z0-9]+){0,2}$`, data.Username)
	if !usernameMatched || err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect username.")
		return
	}

	// Updating a user's username
	if err := h.PostgresDB.UpdateUser(models.Users{Id: userId}, models.Users{Username: data.Username}); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Username updated successfully.",
	})
}

// @Summary   Reset user email
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Param     NewUserEmail  body      models.UserEmail   true  "New user email"
// @Success   200           {object}  models.ApiMessage
// @Failure   400           {object}  models.ApiError
// @Failure   500           {object}  models.ApiError
// @Security  token
// @Router    /user/settings/reset/email [post]
func (h *Handler) ResetUserEmail(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "email": "nktkln@example.com"
		}
	*/

	var data models.UserEmail
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case h.PostgresDB.CheckUserEmail(data.Email):
		NewErrorResponse(c, http.StatusBadRequest, "Wrong email.")
	}
	if c.IsAborted() {
		return
	}

	// Create a temporary access key to verify new email fidelity and then send it
	if err := h.EmailAuthData.UserEmailReset(c.Request.Context(), h.RedisClient, data.Email, userId); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "A verification key was sent to your new email.",
	})
}

// @Summary  Update user email
// @Tags     User settings
// @Accept   json
// @Produce  json
// @Param    key  query     string  true  "Veryfication key"
// @Success  200  {object}  models.ApiMessage
// @Failure  400  {object}  models.ApiError
// @Failure  500  {object}  models.ApiError
// @Router   /user/settings/update/email [patch]
func (h *Handler) UpdateUserEmail(c *gin.Context) {
	// Checking that the key is in working order
	userParam, err := h.RedisClient.GetUserData(c.Request.Context(), c.Query("key"))
	if err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Time has expired, your key is not valid.")
		return
	}

	// Updating a user's email
	if err = h.PostgresDB.UpdateUser(models.Users{Id: userParam.Id}, models.Users{Email: userParam.Email}); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Email successfully updated.",
	})
}

// @Summary  Reset user password
// @Tags     User settings
// @Accept   json
// @Produce  json
// @Param    UserEmail  body      models.UserEmail  true  "User data"
// @Success  200        {object}  models.UserTokens
// @Failure  400        {object}  models.ApiError
// @Failure  500        {object}  models.ApiError
// @Router   /user/settings/reset/password [post]
func (h *Handler) ResetUserPassword(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "email": "nktkln@example.com"
		}
	*/

	var data models.UserEmail

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case h.PostgresDB.CheckUserEmail(data.Email):
		NewErrorResponse(c, http.StatusBadRequest, "Wrong email.")
	}
	if c.IsAborted() {
		return
	}

	// Create a temporary access key to verify email fidelity and then send it
	if err := h.EmailAuthData.UserPasswordReset(c.Request.Context(), h.RedisClient, data.Email); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "A reset key was sent to your email.",
	})
}

// @Summary  Update user password
// @Tags     User settings
// @Accept   json
// @Produce  json
// @Param    NewUserPassword  body      models.UserPassword  true  "User password"
// @Param    key              query    string               true  "Veryfication key"
// @Success  200              {object}  models.UserTokens
// @Failure  404              {object}  models.ApiError
// @Failure  500              {object}  models.ApiError
// @Router   /user/settings/update/password [patch]
func (h *Handler) UpdateUserPassword(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "password": "StRon9Pa$$w0rd"
		}
	*/

	var data models.UserPassword

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case data.Password == "":
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect password.")
	}
	if c.IsAborted() {
		return
	}

	// Checking that the key is in working order
	val, err := h.RedisClient.GetEmailData(c.Request.Context(), c.Query("key"))
	if err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Time has expired, your key is not valid.")
		return
	}

	// Updating a user's password
	if err = h.PostgresDB.UpdateUserPassword(data.Password, val); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Password successfully updated.",
	})
}

// @Summary  Update user token
// @Tags     User settings
// @Accept   json
// @Produce  json
// @Param    refresh_token  header    string  true  "Refresh token"
// @Success  200            {object}  models.UserTokens
// @Router   /user/settings/update/token [put]
func (h *Handler) UpdateUserToken(c *gin.Context) {
	userId := h.RedisClient.VerifyRefreshToken(c.Request.Context(), c.GetHeader("refresh_token"))

	// Input data check
	if userId == 0 {
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
		return
	}

	// Create a new token for the user
	accessToken, refreshToken, err := h.RedisClient.CreateTokens(c.Request.Context(), userId)
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.UserTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// @Summary   Update user icon
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Param     icon  formData  file  true  "User icon"
// @Success   200   {object}  models.ApiMessage
// @Failure   400   {object}  models.ApiError
// @Failure   500   {object}  models.ApiError
// @Security  token
// @Router    /user/settings/update/icon [put]
func (h *Handler) UpdateUserIcon(c *gin.Context) {
	// Setting a limit for the size of the user's icon
	// c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, models.MAX_ICON_UPLOAD_SIZE)

	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	file, fileHeader, err := c.Request.FormFile("icon")

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusBadRequest, "No such file.")
	case fileHeader.Size > models.MAX_ICON_UPLOAD_SIZE:
		NewErrorResponse(c, http.StatusBadRequest, "Icon is too large.")
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	}
	if c.IsAborted() {
		return
	}
	defer file.Close()

	// Checking the file type
	buffer := make([]byte, fileHeader.Size)
	_, err = file.Read(buffer)
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	fileType := http.DetectContentType(buffer)

	if _, ex := models.IMAGE_TYPES[fileType]; !ex {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect user icon file type.")
		return
	}
	
	// Saving a file to the database
	object := models.FileUnit{
		Icon:        bytes.NewReader(buffer),
		Size:        fileHeader.Size,
		ContentType: fileType,
		ID:          userId,
	}

	img, err := h.MinIOClient.UploadFile(c.Request.Context(), object)
	if err != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Problem with uploading a icon to the DB.")
		return
	}

	if err := h.PostgresDB.UpdateUserIcon(userId, img); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "Icon successfully updated.",
	})
}

// @Summary   Delete user icon
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Success   200  {object}  models.ApiMessage
// @Failure   400  {object}  models.ApiError
// @Failure   404  {object}  models.ApiError
// @Failure   500  {object}  models.ApiError
// @Security  token
// @Router    /user/delete/icon [delete]
func (h *Handler) DeleteUserIcon(c *gin.Context) {
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	userData := h.PostgresDB.GetUserById(userId)

	// Input data check
	switch {
	case userData.Id == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case userData.Icon == "":
		NewErrorResponse(c, http.StatusBadRequest, "The user icon is not yet installed.")
	}
	if c.IsAborted() {
		return
	}

	// Delete user icon
	if err := h.MinIOClient.DeleteFile(c.Request.Context(), userData.Icon); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.PostgresDB.UpdateUserIcon(userData.Id, ""); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "The user icon has been deleted.",
	})
}

// @Summary   Delete user account
// @Tags      User settings
// @Accept    json
// @Produce   json
// @Param     password  query     string  true  "User password"
// @Success   200       {object}  models.ApiMessage
// @Failure   400       {object}  models.ApiError
// @Failure   404       {object}  models.ApiError
// @Failure   500       {object}  models.ApiError
// @Security  token
// @Router    /user/delete/account [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	userId := h.RedisClient.VerifyToken(c.Request.Context(), c.GetHeader("token"))
	userData := h.PostgresDB.GetUserById(userId)

	// Input data check
	switch {
	case userId == 0:
		NewErrorResponse(c, http.StatusNotFound, "Inactive user.")
	case bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(c.Query("password"))) != nil:
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect password.")
	}
	if c.IsAborted() {
		return
	}

	// Delete all user data
	if err := h.PostgresDB.DeleteUser(h.MinIOClient, c.Request.Context(), userData); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.RedisClient.DeleteAccessTokensData(c.Request.Context(), userData.Id); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.RedisClient.DeleteRefreshTokensData(c.Request.Context(), userData.Id); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "The account has been deleted.",
	})
}
