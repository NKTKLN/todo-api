package handlers

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/NKTKLN/todo-api/models"
)

// @Summary  Register a user
// @Tags     Authorization
// @Accept   json
// @Produce  json
// @Param    UserData  body      models.UserData  true  "User data"
// @Success  200       {object}  models.ApiMessage
// @Failure  400       {object}  models.ApiError
// @Failure  404       {object}  models.ApiError
// @Failure  500       {object}  models.ApiError
// @Router   /auth/sign-up [post]
func (h *Handler) SignUp(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "email": "nktkln@example.com",
		  "name": "NKTKLN",
		  "password": "StRon9Pa$$w0rd",
		  "username": "nktkln"
		}
	*/

	var data models.UserData

	// Input data check
	switch {
	case c.ShouldBindJSON(&data) != nil:
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
	case data.Email == "" || data.Password == "" || data.Username == "" || data.Name == "":
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect data.")
	case !h.PostgresDB.CheckUserEmail(data.Email):
		NewErrorResponse(c, http.StatusBadRequest, "Mail is already in use.")
	case !h.PostgresDB.CheckUserUsername(data.Username):
		NewErrorResponse(c, http.StatusBadRequest, "Username is already in use.")
	}
	if c.IsAborted() {
		return
	}

	emailMatched, err := regexp.MatchString(`^\w*@\w*[.]\w*$`, data.Email)
	if !emailMatched || err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect email.")
		return
	}
	nameMatched, err := regexp.MatchString(`^[a-zA-Z]+([ ]?[a-zA-Z0-9]+){0,2}$`, data.Name)
	if !nameMatched || err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect name.")
		return
	}
	usernameMatched, err := regexp.MatchString(`^[a-z]+([-_]?[a-z0-9]+){0,2}$`, data.Username)
	if !usernameMatched || err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Incorrect username.")
		return
	}

	// Sending an email verification code to a user
	if err := h.EmailAuthData.UserEmailVerification(c.Request.Context(), h.RedisClient, data); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.ApiMessage{
		Message: "A verification key was sent to your email."},
	)
}

// @Summary   Confirm the new user's email
// @Tags      Authorization
// @Accept    json
// @Produce   json
// @Param     key  query     string  true  "Verification key"
// @Success   200  {object}  models.UserTokens
// @Failure   400  {object}  models.ApiError
// @Failure   500  {object}  models.ApiError
// @Router    /auth/verify [get]
func (h *Handler) VerifySignUp(c *gin.Context) {
	userParam, err := h.RedisClient.GetUserData(c.Request.Context(), c.Query("key"))

	// Input data check
	switch {
	case err != nil:
		NewErrorResponse(c, http.StatusBadRequest, "Time has expired, your key is not valid.")
	case !h.PostgresDB.CheckUserEmail(userParam.Email):
		NewErrorResponse(c, http.StatusBadRequest, "Mail is already in use.")
	case !h.PostgresDB.CheckUserUsername(userParam.Username):
		NewErrorResponse(c, http.StatusBadRequest, "Username is already in use.")
	}
	if c.IsAborted() {
		return
	}

	// Adding a new user to the database
	userId, err := h.PostgresDB.CrateUser(userParam)
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Creating new token for user
	accessToken, refreshToken, err := h.RedisClient.CreateTokens(c.Request.Context(), userId)
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.RedisClient.DeleteEmailData(c.Request.Context(), c.Query("key")); err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.UserTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// @Summary  Sign in to your account
// @Tags     Authorization
// @Accept   json
// @Produce  json
// @Param    LoginData  body      models.UserLoginData  true  "User data"
// @Success  200        {object}  models.UserTokens
// @Failure  400        {object}  models.ApiError
// @Failure  500        {object}  models.ApiError
// @Router   /auth/sign-in [post]
func (h *Handler) SignIn(c *gin.Context) {
	/*
		Example of JSON received

		{
		  "email": "nktkln@example.com",
		  "password": "StRon9Pa$$w0rd"
		}
	*/

	var data models.UserLoginData

	// Input data check
	if c.ShouldBindJSON(&data) != nil {
		NewErrorResponse(c, http.StatusInternalServerError, "Data retrieval error.")
		return
	}

	// User data check
	userData := h.PostgresDB.GetUserByEmail(data.Email)
	if bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(data.Password)) != nil {
		NewErrorResponse(c, http.StatusBadRequest, "Wrong email or password.")
		return
	}

	accessToken, refreshToken, err := h.RedisClient.CheckTokens(c.Request.Context(), userData.Id)
	if err != nil {
		NewServerErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.UserTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
