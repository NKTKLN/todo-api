package db

import (
	"context"

	"github.com/minio/minio-go/v7"

	"github.com/NKTKLN/todo-api/models"
)

type PostgresDB interface {
	UserOperations
	ListOperations
	TaskOperations
	SubtaskOperations
}

type RedisClient interface {
	EmailOperations
	TokenOperations
}

type MinIOClient interface {
	Connect() error
	CreateBucket(context.Context) error
	UploadFile(context.Context, models.FileUnit) (string, error)
	DownloadFile(context.Context, string) (*minio.Object, error)
	DeleteFile(context.Context, string) error
}

// Postgres operations
type UserOperations interface {
	CrateUser(models.Users) (int, error)
	GetUserByEmail(string) models.Users
	CheckUserUsername(string) bool
	CheckUserEmail(string) bool
	GetUserById(int) models.Users
	UpdateUser(models.Users, models.Users) error
	UpdateUserPassword(string, string) error
	UpdateUserIcon(int, string) error
	CheckUserPassword(string, string) error
	DeleteUser(MinIOClient, context.Context, models.Users) error
}

type ListOperations interface {
	CreateList(models.Lists) error
	GetAllUserLists(int) []models.ListsData
	GetListsForEditIndex(int, int) []models.Lists
	GetListById(int) models.Lists
	GetListByIdAndUserId(int, int) models.Lists
	GetListMaxIndex(int) int
	UpdateListData(models.Lists) error
	UpdateListIndex(int, int) error
	UpdateListsIndexes(models.Lists) error
	DeleteList(int) error
}

type TaskOperations interface {
	CreateTask(models.Tasks) error
	GetAllTasks(int) []models.TasksData
	GetTaskById(int) models.Tasks
	GetTasksForEditIndex(int, int) []models.Tasks
	GetListIdWhereTask(int, int) int
	GetTaskMaxIndex(int) int
	UpdateTaskData(models.Tasks) error
	UpdateTaskIndex(int, int) error
	UpdateTasksIndexes(models.Tasks) error
	DeleteTask(int) error
}

type SubtaskOperations interface {
	CreateSubtask(models.Tasks) error
	GetAllSubtasks(int) []models.SubtasksData
	GetSubtasksForEditIndex(int, int) []models.Tasks
	GetTaskIdWhereSubtask(int) int
	GetSubtaskMaxIndex(int) int
	UpdateSubtasksIndexes(models.Tasks) error
	DeleteSubtask(int) error
}

// Redis operations
type EmailOperations interface {
	AddEmailData(context.Context, interface{}) (string, error)
	GetEmailData(context.Context, string) (string, error)
	GetUserData(context.Context, string) (models.Users, error)
	DeleteEmailData(context.Context, string) error
}

type TokenOperations interface {
	VerifyToken(context.Context, string) int
	VerifyRefreshToken(context.Context, string) int
	CreateTokens(context.Context, int) (string, string, error)
	CheckTokens(context.Context, int) (string, string, error)
	DeleteRefreshTokensData(context.Context, int) error
	DeleteAccessTokensData(context.Context, int) error
}

