package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/common"
	rd "github.com/NKTKLN/todo-api/pkg/db/redis"
	"github.com/NKTKLN/todo-api/pkg/handlers"
)

var _ = Describe("Subtask", func() {
	var (
		r                       *gin.Engine
		w                       *httptest.ResponseRecorder
		accessJwt               string
		handler                 handlers.Handler
		postgresMock            sqlmock.Sqlmock
		redisClientAccessToken  *redis.Client
		redisClientRefreshToken *redis.Client
	)

	BeforeEach(func() {
		gin.SetMode(gin.ReleaseMode)

		r = gin.New()
		w = httptest.NewRecorder()

		redisClientAccessToken = TestRedisConnection()
		redisClientRefreshToken = TestRedisConnection()

		handler.RedisClient = &rd.RedisClients{
			AccessTokenClient:  redisClientAccessToken,
			RefreshTokenClient: redisClientRefreshToken,
		}

		handler.PostgresDB, postgresMock = MockPostgresConnection()

		// Generate new jwt token
		accessJwt, _ = common.NewJWT(117115101114, time.Minute, viper.GetString("api.jwt.access-secret"))

		// Adding data to redis
		redisClientAccessToken.Set(context.Background(), "117115101114", accessJwt, time.Minute)
	})

	AfterEach(func() {
		redisClientAccessToken.Close()
		redisClientRefreshToken.Close()

		Expect(postgresMock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
	})

	Describe("Add subtask", func() {
		BeforeEach(func() {
			r.POST("/todo/subtask/add", handler.AddSubtask)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"comment": "Test Subtask Comment", "task_id": 11697115107, "name": "Test Subtask Name"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("task not found", func() {
			const requestBody = `{"comment": "Test Subtask Comment", "task_id": 11697115107, "name": "Test Subtask Name"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the task not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This task not found."}`))
			})
		})

		Describe("Incorrect name", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))
			})

			Context("empty name", func() {
				const requestBody = `{"comment": "Test Subtask Comment", "task_id": 11697115107, "name": ""}`

				BeforeEach(func() {
					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return an error that the name is empty", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Empty name."}`))
				})
			})

			Context("name longer than 32 characters", func() {
				const requestBody = `{"comment": "Test Subtask Comment", "task_id": 11697115107, "name": "a very complicated and long name for the list"}`

				BeforeEach(func() {
					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return an error that the name longer than 32 characters", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"A name longer than 32 characters."}`))
				})
			})
		})

		Describe("Ok", func() {
			const requestBody = `{"comment": "Test Subtask Comment", "task_id": 11697115107, "name": "Test Subtask Name"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
					WithArgs(AnyInt{}).
					WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))
			})

			Context("the user does not have any tasks in the list yet", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksByTaskId)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlInsertTaskData)).
						WithArgs(0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, AnyTime{}, false, false, AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(0))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the task was successfully created", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Subtask added to db."}`))
				})
			})

			Context("the user has tasks in the list", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksByTaskId)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Suntask Name", "Test Suntask Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxSubtaskIndex)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(0))

					postgresMock.ExpectBegin()
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlInsertTaskData)).
						WithArgs(0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 1, nil, AnyTime{}, false, false, AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(0))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/subtask/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the task was successfully created", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Subtask added to db."}`))
				})
			})
		})
	})

	Describe("Delete task", func() {
		BeforeEach(func() {
			r.DELETE("/todo/subtask/delete", handler.DeleteSubtask)
		})

		Context("error when converting subtask_id", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/todo/subtask/delete?subtask_id="1151179811697115107"`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error message when converting the subtask_id", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Error when converting subtask_id."}`))
			})
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(11697115107))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(0, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/todo/subtask/delete?subtask_id=1151179811697115107`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("this subtask not found", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, "/todo/subtask/delete?subtask_id=1151179811697115107", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the subtask is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This subtask not found."}`))
			})
		})

		Describe("Ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(11697115107))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))
			})

			Context("deleting a single subtask", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksForEditIndex)).
						WithArgs(11697115107, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteTask)).
						WithArgs(1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodDelete, "/todo/subtask/delete?subtask_id=1151179811697115107", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the subtask was successfully deleted", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"The subtask has been deleted."}`))
				})
			})

			Context("deleting a subtasks with changing the index of other subtasks", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksForEditIndex)).
						WithArgs(11697115107, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115108, 0, 11697115107, "Test Subtask Name 2", "Test Subtask Comment 2", 1, nil, nil, false, false))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(0, 1151179811697115108).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteTask)).
						WithArgs(1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodDelete, "/todo/subtask/delete?subtask_id=1151179811697115107", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the task was successfully deleted", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"The subtask has been deleted."}`))
				})
			})
		})
	})

	Describe("Edit subtasks", func() {
		BeforeEach(func() {
			r.PATCH("/todo/subtask/edit", handler.EditSubtask)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"end_time": "2077-12-10 13:13", "id": 1151179811697115107}`

			BeforeEach(func() {
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(0, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("this task not found", func() {
			const requestBody = `{"end_time": "2077-12-10 13:13", "id": 1151179811697115107}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the subtask is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This subtask not found."}`))
			})
		})

		Describe("Incorrect data", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(11697115107))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))
			})

			Describe("Incorrect time", func() {
				Context("incorrect time format", func() {
					const requestBody = `{"end_time": "", "id": 1151179811697115107}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return a message that the time format is incorrect", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Incorrect time format."}`))
					})
				})

				Context("incorrect time", func() {
					const requestBody = `{"end_time": "0001-01-01 00:00", "id": 1151179811697115107}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return a message that the time is incorrect", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Incorrect time."}`))
					})
				})
			})

			Describe("Incorrect name", func() {
				Context("empty name", func() {
					const requestBody = `{"name": "", "end_time": "2077-12-10 13:13", "id": 1151179811697115107}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the name is empty", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Empty name."}`))
					})
				})

				Context("name longer than 32 characters", func() {
					const requestBody = `{"name": "a very complicated and long name for the subtask", "end_time": "2077-12-10 13:13", "id": 1151179811697115107}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the name longer than 32 characters", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"A name longer than 32 characters."}`))
					})
				})
			})

			Describe("Incorrect index", func() {
				Context("index is greater than the maximum index", func() {
					const requestBody = `{"name": "Test Subtask Name", "end_time": "2077-12-10 13:13", "id": 1151179811697115107, "index": 1}`

					BeforeEach(func() {
						// Query building for the postgres
						postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxSubtaskIndex)).
							WithArgs(11697115107).
							WillReturnRows(sqlmock.NewRows([]string{"index"}).
								AddRow(0))

						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, `/todo/subtask/edit`, bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the index is incorrect", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Incorrect index."}`))
					})
				})

				Context("index is less than the null", func() {
					const requestBody = `{"name": "Test Subtask Name", "end_time": "2077-12-10 13:13", "id": 1151179811697115107, "index": -1}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, `/todo/subtask/edit`, bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the index is incorrect", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Incorrect index."}`))
					})
				})
			})
		})

		Describe("Ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskIdBySubtaskId)).
					WithArgs(1151179811697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(11697115107))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))
			})

			Context("update the subtask without changing the index", func() {
				const requestBody = `{"name": "Test Subtask Name", "comment": "Test Subtask Comment", "end_time": "2077-12-10 13:13", "id": 1151179811697115107, "index": 0}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxSubtaskIndex)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTask)).
						WithArgs("Test Subtask Name", "Test Subtask Comment", nil, AnyTime{}, false, false, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the task data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the task data was successful."}`))
				})
			})

			Context("update the task with changing the index", func() {
				const requestBody = `{"name": "Test Subtask Name", "comment": "Test Subtask Comment", "end_time": "2077-12-10 13:13", "id": 1151179811697115107, "index": 1}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxSubtaskIndex)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTask)).
						WithArgs("Test Subtask Name", "Test Subtask Comment", nil, AnyTime{}, false, false, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksToIncreaseTheIndex)).
						WithArgs(11697115107, 1, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false).
							AddRow(1151179811697115108, 0, 11697115107, "Test Subtask Name 2", "Test Subtask Comment 2", 1, nil, nil, false, false))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(-1, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(0, 1151179811697115108).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(1, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the task data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the task data was successful."}`))
				})
			})

			Context("update the task with changing the index", func() {
				const requestBody = `{"name": "Test Subtask Name", "comment": "Test Subtask Comment", "end_time": "2077-12-10 13:13", "id": 1151179811697115107, "index": 0}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxSubtaskIndex)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTask)).
						WithArgs("Test Subtask Name", "Test Subtask Comment", nil, AnyTime{}, false, false, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 1, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectTaskById)).
						WithArgs(1151179811697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 1, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksForIndexReduction)).
						WithArgs(11697115107, 0, 1).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 1, nil, nil, false, false).
							AddRow(1151179811697115108, 0, 11697115107, "Test Subtask Name 2", "Test Subtask Comment 2", 0, nil, nil, false, false))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(2, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(1, 1151179811697115108).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditTaskIndex)).
						WithArgs(0, 1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, "/todo/subtask/edit", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the task data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the task data was successful."}`))
				})
			})
		})
	})

	Describe("Show tasks", func() {
		BeforeEach(func() {
			r.GET("/todo/subtask/show", handler.ShowSubtasks)
		})

		Context("error when converting task_id", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/todo/subtask/show?task_id="11697115107"`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error message when converting the task_id", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Error when converting task_id."}`))
			})
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/todo/subtask/show?task_id=11697115107`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("this list not found", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, "/todo/subtask/show?task_id=11697115107", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the list is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This list not found."}`))
			})
		})

		Describe("Ok", func() {
			var subtasks models.ApiShowSubtasks

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListIdWhereTask)).
					WithArgs(117115101114, 11697115107).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(108105115116))
			})

			Context("without tasks", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksByTaskId)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/todo/subtask/show?task_id=11697115107", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &subtasks)).To(BeNil())
				})

				It("should return empty task data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(subtasks.Subtasks).To(BeNil())
				})
			})

			Context("with tasks", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksByTaskId)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Subtask Name", "Test Subtask Comment", 0, nil, nil, false, false))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/todo/subtask/show?task_id=11697115107", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &subtasks)).To(BeNil())
				})

				It("should return task data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(subtasks.Subtasks).To(Equal([]models.SubtasksData{{Id: 1151179811697115107, Name: "Test Subtask Name", Comment: "Test Subtask Comment", Index: 0, EndTime: "0001-01-01 00:00"}}))
				})
			})
		})
	})
})
