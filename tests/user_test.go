package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	"github.com/NKTKLN/todo-api/pkg/db/minio"
	rd "github.com/NKTKLN/todo-api/pkg/db/redis"
	"github.com/NKTKLN/todo-api/pkg/handlers"
)

var _ = Describe("User", func() {
	var (
		r            *gin.Engine
		w            *httptest.ResponseRecorder
		handler      handlers.Handler
		postgresMock sqlmock.Sqlmock
	)

	BeforeEach(func() {
		gin.SetMode(gin.ReleaseMode)

		r = gin.New()
		w = httptest.NewRecorder()

		handler.PostgresDB, postgresMock = MockPostgresConnection()
	})

	AfterEach(func() {
		Expect(postgresMock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
	})

	Describe("Show user icon", func() {
		BeforeEach(func() {
			r.GET("/user/show/icon", handler.GetUserIcon)
		})

		Context("error when converting user_id", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/icon?user_id="117115101114"`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error message when converting the user_id", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Error when converting user_id."}`))
			})
		})

		Context("user not found", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/icon?user_id=117115101114`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"User not found."}`))
			})
		})

		Context("the user icon is not yet installed", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/icon?user_id=117115101114`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user icon is not installed yet", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"The user icon is not yet installed."}`))
			})
		})

		Context("ok", func() {
			var imageBytes, imageData []byte

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", "user-117115101114.png"))

				// Connecting to the minio db
				handler.MinIOClient = minio.NewMinioProvider(
					"play.min.io",
					"Q3AM3UQ867SPQQA43P2F",
					"zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
					true,
				)
				err := handler.MinIOClient.Connect()
				Expect(err).To(BeNil())

				// Adding an icon to the minio db
				file, err := os.Open("static/test_icon.png")
				Expect(err).To(BeNil())
				fInfo, err := file.Stat()
				Expect(err).To(BeNil())
				defer file.Close()

				_, err = handler.MinIOClient.UploadFile(context.Background(), models.FileUnit{
					Icon:        file,
					Size:        fInfo.Size(),
					ContentType: "image/png",
					ID:          117115101114,
				})
				Expect(err).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/icon?user_id=117115101114`, nil)
				r.ServeHTTP(w, req)

				// Converting the image to []bytes
				imageBytes = make([]byte, fInfo.Size())
				buffer := bufio.NewReader(file)
				_, err = buffer.Read(imageBytes)
				Expect(err).To(BeNil())

				// Retrieving an image from the API
				imageData, err = io.ReadAll(w.Body)
				Expect(err).To(BeNil())
			})

			It("should return the user icon", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(imageData).To(Equal(imageBytes))
			})
		})
	})

	Describe("Show user data by id", func() {
		BeforeEach(func() {
			r.GET("/user/show/data-by-id", handler.GetUserData)
		})

		Context("error when converting user_id", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/data-by-id?user_id="117115101114"`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error message when converting the user_id", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Error when converting user_id."}`))
			})
		})

		Context("user not found", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/data-by-id?user_id=117115101114`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"User not found."}`))
			})
		})

		Context("ok", func() {
			var user models.ShowUserData

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/data-by-id?user_id=117115101114`, nil)
				r.ServeHTTP(w, req)

				// Converting the query body into a model
				Expect(json.Unmarshal(w.Body.Bytes(), &user)).To(BeNil())
			})

			It("should return the user data", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(user).To(Equal(models.ShowUserData{
					Id:       117115101114,
					Name:     "Test User Name",
					Username: "test_username",
				}))
			})
		})
	})

	Describe("Show user data by token", func() {
		var (
			accessJwt              string
			redisClientAccessToken *redis.Client
		)

		BeforeEach(func() {
			redisClientAccessToken = TestRedisConnection()

			handler.RedisClient = &rd.RedisClients{
				AccessTokenClient: redisClientAccessToken,
			}

			// Generate new jwt token
			accessJwt, _ = common.NewJWT(117115101114, time.Minute, viper.GetString("api.jwt.access-secret"))

			// Adding data to redis
			redisClientAccessToken.Set(context.Background(), "117115101114", accessJwt, time.Minute)

			r.GET("/user/show/data-by-token", handler.GetUserDataByToken)
		})

		AfterEach(func() {
			redisClientAccessToken.Close()
		})

		Context("user not found", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/data-by-token`, nil)
				req.Header.Add("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is not found", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"User not found."}`))
			})
		})

		Context("ok", func() {
			var user models.ShowUserData

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, `/user/show/data-by-token`, nil)
				req.Header.Add("token", accessJwt)
				r.ServeHTTP(w, req)

				// Converting the query body into a model
				Expect(json.Unmarshal(w.Body.Bytes(), &user)).To(BeNil())
			})

			It("should return the user data", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(user).To(Equal(models.ShowUserData{
					Id:       117115101114,
					Name:     "Test User Name",
					Username: "test_username",
				}))
			})
		})
	})
})
