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

var _ = Describe("Lists", func() {
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

	Describe("Add list", func() {
		BeforeEach(func() {
			r.POST("/todo/list/add", handler.AddList)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/todo/list/add", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"comment": "Test List Comment", "name": "Test List Name"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/todo/list/add", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Describe("Incorrect name", func() {
			Context("empty name", func() {
				const requestBody = `{"comment": "Test List Comment", "name": ""}`

				BeforeEach(func() {
					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/list/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return an error that the name is empty", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Empty name."}`))
				})
			})

			Context("name longer than 32 characters", func() {
				const requestBody = `{"comment": "Test List Comment", "name": "a very complicated and long name for the list"}`

				BeforeEach(func() {
					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/list/add", bytes.NewBufferString(requestBody))
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
			const requestBody = `{"comment": "Test List Comment", "name": "Test List Name"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
					WithArgs(AnyInt{}).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))
			})

			Context("the user has no lists yet", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsByUserId)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlInsertListData)).
						WithArgs(117115101114, "Test List Name", "Test List Comment", 0, AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(0))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/list/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the list was successfully created", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"List added to db."}`))
				})
			})

			Context("the user already has lists", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsByUserId)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxListIndex)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(0))

					postgresMock.ExpectBegin()
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlInsertListData)).
						WithArgs(117115101114, "Test List Name", "Test List Comment", 1, AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(0))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/todo/list/add", bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the list was successfully created", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"List added to db."}`))
				})
			})
		})
	})

	Describe("Delete list", func() {
		BeforeEach(func() {
			r.DELETE("/todo/list/delete", handler.DeleteList)
		})

		Context("error when converting list_id", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(0, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/todo/list/delete?list_id="108105115116"`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error message when converting the list_id", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Error when converting list_id."}`))
			})
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/todo/list/delete?list_id=108105115116`, nil)
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
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, "/todo/list/delete?list_id=108105115116", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the list is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This list not found."}`))
			})
		})

		Describe("Ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
						AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))
			})

			Context("deleting a single list without tasks", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsForEditIndex)).
						WithArgs(117115101114, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllTasksByListId)).
						WithArgs(AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteList)).
						WithArgs(108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodDelete, "/todo/list/delete?list_id=108105115116", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the list was successfully deleted", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"The list has been deleted."}`))
				})
			})

			Context("deleting a list without tasks with changing the index of other lists", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsForEditIndex)).
						WithArgs(117115101114, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(1081051151162, 117115101114, "Test List Name 2", "Test List Comment 2", 1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(0, 1081051151162).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllTasksByListId)).
						WithArgs(AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteList)).
						WithArgs(108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodDelete, "/todo/list/delete?list_id=108105115116", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the list was successfully deleted", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"The list has been deleted."}`))
				})
			})

			Context("deleting a single list with tasks and subtasks", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsForEditIndex)).
						WithArgs(117115101114, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllTasksByListId)).
						WithArgs(AnyInt{}).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(11697115107, 108105115116, 0, "Test Task Name", "Test Task Comment", 0, nil, nil, false, false))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllSubtasksByTaskId)).
						WithArgs(11697115107).
						WillReturnRows(sqlmock.NewRows([]string{"id", "list_id", "task_id", "name", "comment", "index", "categories", "end_time", "done", "special"}).
							AddRow(1151179811697115107, 0, 11697115107, "Test Task Name", "Test Task Comment", 0, nil, nil, false, false))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteTask)).
						WithArgs(1151179811697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteTask)).
						WithArgs(11697115107).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteList)).
						WithArgs(108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodDelete, "/todo/list/delete?list_id=108105115116", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message that the list was successfully deleted", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"The list has been deleted."}`))
				})
			})
		})
	})

	Describe("Edit list", func() {
		BeforeEach(func() {
			r.PATCH("/todo/list/edit", handler.EditList)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/todo/list/edit", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": "Test List Name"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("this list not found", func() {
			const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": "Test List Name"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the list not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"This list not found."}`))
			})
		})

		Describe("Incorrect data", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
						AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))
			})

			Describe("Incorrect name", func() {
				Context("empty name", func() {
					const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": ""}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/list/edit", bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the name is empty", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Empty name."}`))
					})
				})

				Context("name longer than 32 characters", func() {
					const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": "a very complicated and long name for the list"}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, "/todo/list/edit", bytes.NewBufferString(requestBody))
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
					const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 1, "name": "Test List Name"}`

					BeforeEach(func() {
						// Query building for the postgres
						postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxListIndex)).
							WithArgs(117115101114).
							WillReturnRows(sqlmock.NewRows([]string{"index"}).
								AddRow(0))

						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
						req.Header.Set("token", accessJwt)
						r.ServeHTTP(w, req)
					})

					It("should return an error that the index is incorrect", func() {
						Expect(w.Code).To(Equal(http.StatusBadRequest))
						Expect(w.Body.String()).To(Equal(`{"error":"Incorrect index."}`))
					})
				})

				Context("index is less than the null", func() {
					const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": -1, "name": "Test List Name"}`

					BeforeEach(func() {
						// Sending a query with data
						req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
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
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListByIdAndUserId)).
					WithArgs(108105115116, 117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
						AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))
			})

			Context("update the list without changing the index", func() {
				const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": "Test List Name"}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxListIndex)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(0))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditList)).
						WithArgs("Test List Name", "Test List Comment", 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
						WithArgs(108105115116).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the list data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the list data was successful."}`))
				})
			})

			Context("update the list with changing the index", func() {
				const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 1, "name": "Test List Name"}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxListIndex)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditList)).
						WithArgs("Test List Name", "Test List Comment", 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
						WithArgs(108105115116).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
						WithArgs(108105115116).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsToIncreaseTheIndex)).
						WithArgs(117115101114, 1, 0).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0).
							AddRow(1081051151162, 117115101114, "Test List Name", "Test List Comment", 1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(-1, 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(0, 1081051151162).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(1, 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the list data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the list data was successful."}`))
				})
			})

			Context("update the list with changing the index", func() {
				const requestBody = `{"comment": "Test List Comment", "id": 108105115116, "index": 0, "name": "Test List Name"}`

				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectMaxListIndex)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"index"}).
							AddRow(1))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditList)).
						WithArgs("Test List Name", "Test List Comment", 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
						WithArgs(108105115116).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 1))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectListById)).
						WithArgs(108105115116).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 1))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsForIndexReduction)).
						WithArgs(117115101114, 0, 1).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 1).
							AddRow(1081051151162, 117115101114, "Test List Name", "Test List Comment", 0))

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(2, 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(1, 1081051151162).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					postgresMock.ExpectBegin()
					postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditListIndex)).
						WithArgs(0, 108105115116).
						WillReturnResult(sqlmock.NewResult(1, 1))
					postgresMock.ExpectCommit()

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPatch, `/todo/list/edit`, bytes.NewBufferString(requestBody))
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)
				})

				It("should return a message about successful update of the list data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Body.String()).To(Equal(`{"message":"Updating the list data was successful."}`))
				})
			})
		})
	})

	Describe("Show lists", func() {
		BeforeEach(func() {
			r.GET("/todo/list/show", handler.ShowLists)
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/todo/list/show`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Describe("Ok", func() {
			var lists models.ApiShowLists

			Context("without lists", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsByUserId)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/todo/list/show", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &lists)).To(BeNil())
				})

				It("should return empty list data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(lists.Lists).To(BeNil())
				})
			})

			Context("with lists", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsByUserId)).
						WithArgs(117115101114).
						WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
							AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/todo/list/show", nil)
					req.Header.Set("token", accessJwt)
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &lists)).To(BeNil())
				})

				It("should return list data", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(lists.Lists).To(Equal([]models.ListsData{{Id: 108105115116, Name: "Test List Name", Comment: "Test List Comment", Index: 0}}))
				})
			})
		})
	})
})
