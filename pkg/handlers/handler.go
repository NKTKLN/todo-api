package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/NKTKLN/todo-api/docs"
	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/common"
	"github.com/NKTKLN/todo-api/pkg/db"
)

type Handler struct {
	PostgresDB    db.PostgresDB
	RedisClient   db.RedisClient
	MinIOClient   db.MinIOClient
	EmailAuthData common.EmailProvider
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth := r.Group("/auth")
	{
		auth.POST("/sign-up", h.SignUp)
		auth.GET("/verify", h.VerifySignUp)
		auth.POST("/sign-in", h.SignIn)
	}

	user := r.Group("/user")
	{
		settigns := user.Group("/settings")
		{
			update := settigns.Group("/update")
			{
				update.PATCH("/name", h.EditUserName)
				update.PATCH("/username", h.EditUserUsername)
				update.PATCH("/email", h.UpdateUserEmail)
				update.PATCH("/password", h.UpdateUserPassword)
				update.PUT("/token", h.UpdateUserToken)
				update.PUT("/icon", h.UpdateUserIcon)
			}
		}

		showData := user.Group("/show")
		{
			showData.GET("/icon", h.GetUserIcon)
			showData.GET("/data-by-id", h.GetUserData)
			showData.GET("/data-by-token", h.GetUserDataByToken)
		}

		deleteData := user.Group("/delete")
		{
			deleteData.DELETE("/icon", h.DeleteUserIcon)
			deleteData.DELETE("/account", h.DeleteUser)
		}
	}

	todo := r.Group("/todo")
	{
		list := todo.Group("/list")
		{
			list.POST("/add", h.AddList)
			list.DELETE("/delete", h.DeleteList)
			list.PUT("/edit", h.EditList)
			list.GET("/show", h.ShowLists)
		}

		task := todo.Group("/task")
		{
			task.POST("/add", h.AddTask)
			task.DELETE("/delete", h.DeleteTask)
			task.PUT("/edit", h.EditTask)
			task.GET("/show", h.ShowTasks)
		}

		subtask := todo.Group("/subtask")
		{
			subtask.POST("/add", h.AddSubtask)
			subtask.DELETE("/delete", h.DeleteSubtask)
			subtask.PUT("/edit", h.EditSubtask)
			subtask.GET("/show", h.ShowSubtasks)
		}
	}

	return r
}

func NewErrorResponse(c *gin.Context, statusCode int, message string) {
	c.AbortWithStatusJSON(statusCode, models.ApiError{Error: message})
}

func NewServerErrorResponse(c *gin.Context, statusCode int, message string) {
	logrus.Error(message)
	c.AbortWithStatusJSON(statusCode, models.ApiError{Error: message})
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, token, key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
