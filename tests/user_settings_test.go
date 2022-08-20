package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/common"
	"github.com/NKTKLN/todo-api/pkg/db/minio"
	rd "github.com/NKTKLN/todo-api/pkg/db/redis"
	"github.com/NKTKLN/todo-api/pkg/handlers"
)

func filePreparation(path string) (*bytes.Buffer, *multipart.Writer, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("icon", path)
	if err != nil {
		return nil, nil, err
	}

	sample, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	_, err = io.Copy(part, sample)
	if err != nil {
		return nil, nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}

	return body, writer, nil
}

var _ = Describe("User settings", func() {
	var (
		r                       *gin.Engine
		w                       *httptest.ResponseRecorder
		accessJwt               string
		handler                 handlers.Handler
		postgresMock            sqlmock.Sqlmock
		redisClientEmail        *redis.Client
		redisClientAccessToken  *redis.Client
		redisClientRefreshToken *redis.Client
	)

	BeforeEach(func() {
		gin.SetMode(gin.ReleaseMode)

		r = gin.New()
		w = httptest.NewRecorder()

		redisClientEmail = TestRedisConnection()
		redisClientAccessToken = TestRedisConnection()
		redisClientRefreshToken = TestRedisConnection()

		handler.EmailAuthData = NewFakeEmailProvider("email@example.com", "StRon9Pa$$w0rd", "smtp.example.com", 0)

		handler.RedisClient = &rd.RedisClients{
			EmailClient:        redisClientEmail,
			AccessTokenClient:  redisClientAccessToken,
			RefreshTokenClient: redisClientRefreshToken,
		}

		handler.PostgresDB, postgresMock = MockPostgresConnection()

		handler.MinIOClient = minio.NewMinioProvider(
			"play.min.io",
			"Q3AM3UQ867SPQQA43P2F",
			"zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
			true,
		)
		if err := handler.MinIOClient.Connect(); err != nil {
			logrus.Fatalf("error when connecting to the MinIO database: %s", err.Error())
		}

		// Generate new jwt token
		accessJwt, _ = common.NewJWT(117115101114, time.Minute, viper.GetString("api.jwt.access-secret"))

		// Adding data to redis
		redisClientAccessToken.Set(context.Background(), "117115101114", accessJwt, time.Minute)
	})

	AfterEach(func() {
		redisClientEmail.Close()
		redisClientAccessToken.Close()
		redisClientRefreshToken.Close()

		Expect(postgresMock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
	})

	Describe("Edit user name", func() {
		BeforeEach(func() {
			r.PATCH("/user/settings/update/name", handler.EditUserName)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/name", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"name": "Test User Name"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/name", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("incorrect name", func() {
			const requestBody = `{"name": "Incorrect User Name ❤️"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/name", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user name is incorrect", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect name."}`))
			})
		})

		Context("ok", func() {
			const requestBody = `{"name": "Test User Name"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserName)).
					WithArgs("Test User Name", 117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/name", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user name was updated successfully", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"Name updated successfully."}`))
			})
		})
	})

	Describe("Edit user username", func() {
		BeforeEach(func() {
			r.PATCH("/user/settings/update/username", handler.EditUserUsername)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/username", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"username": "test_username"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/username", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("this username is already in use", func() {
			const requestBody = `{"username": "test_username"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
					WithArgs("test_username").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/username", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the username is occupied", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"This username is already in use."}`))
			})
		})

		Context("incorrect username", func() {
			const requestBody = `{"username": "incorrect__username"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
					WithArgs("incorrect__username").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/username", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the username is occupied", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect username."}`))
			})
		})

		Context("ok", func() {
			const requestBody = `{"username": "test_username"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
					WithArgs("test_username").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserUsername)).
					WithArgs("test_username", 117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/username", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user username was updated successfully", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"Username updated successfully."}`))
			})
		})
	})

	Describe("Reset user email", func() {
		BeforeEach(func() {
			r.POST("/user/settings/reset/email", handler.ResetUserEmail)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/email", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("inactive user", func() {
			const requestBody = `{"email": "email@example.com"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/email", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("wrong email", func() {
			const requestBody = `{"email": "email@example.com"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/email", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should give an error saying that the mail is wrong", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Wrong email."}`))
			})
		})

		Context("ok", func() {
			const requestBody = `{"email": "email@example.com"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/email", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that a confirmation code was sent to the mail", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"A verification key was sent to your new email."}`))
			})
		})
	})

	Describe("Update user email", func() {
		BeforeEach(func() {
			r.PATCH("/user/settings/update/email", handler.UpdateUserEmail)
		})

		Context("time has expired, your key is not valid", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/email?key=key", nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the key is invalid", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Time has expired, your key is not valid."}`))
			})
		})

		Context("ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserEmail)).
					WithArgs("email@example.com", 117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

				// Convert data to json
				jsonData, err := json.Marshal(models.Users{Id: 117115101114, Email: "email@example.com"})
				Expect(err).To(BeNil())

				// Adding data to redis
				redisClientEmail.Set(context.Background(), "key", jsonData, time.Minute)

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/email?key=key", nil)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user's mail has been successfully updated", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"Email successfully updated."}`))
			})
		})
	})

	Describe("Reset user password", func() {
		BeforeEach(func() {
			r.POST("/user/settings/reset/password", handler.ResetUserPassword)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/password", nil)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("wrong email", func() {
			const requestBody = `{"email": "email@example.com"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/password", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should give an error saying that the mail is wrong", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Wrong email."}`))
			})
		})

		Context("ok", func() {
			const requestBody = `{"email": "email@example.com"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/user/settings/reset/password", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return a message that a confirmation code was sent to the mail", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"A reset key was sent to your email."}`))
			})
		})
	})

	Describe("Update user password", func() {
		BeforeEach(func() {
			r.PATCH("/user/settings/update/password", handler.UpdateUserPassword)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/password", nil)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("incorrect password", func() {
			const requestBody = `{"password":""}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/password", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the password is occupied", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect password."}`))
			})
		})

		Context("time has expired, your key is not valid", func() {
			const requestBody = `{"password":"StRon9Pa$$w0rd"}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/password?key=key", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the key is invalid", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Time has expired, your key is not valid."}`))
			})
		})

		Context("ok", func() {
			const requestBody = `{"password":"StRon9Pa$$w0rd"}`

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserPassword)).
					WithArgs(AnyString{}, "email@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

				// Adding data to redis
				redisClientEmail.Set(context.Background(), "key", "email@example.com", time.Minute)

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPatch, "/user/settings/update/password?key=key", bytes.NewBufferString(requestBody))
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user's password has been successfully updated", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"Password successfully updated."}`))
			})
		})
	})

	Describe("Update user token", func() {
		BeforeEach(func() {
			r.PUT("/user/settings/update/token", handler.UpdateUserToken)
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/token", nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("ok", func() {
			var tokens models.UserTokens

			BeforeEach(func() {
				// Generate new jwt token
				refreshJwt, _ := common.NewJWT(117115101114, time.Minute, viper.GetString("api.jwt.refresh-secret"))

				// Adding data to redis
				redisClientRefreshToken.Set(context.Background(), "117115101114", refreshJwt, time.Minute)

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/token", nil)
				req.Header.Set("refresh_token", refreshJwt)
				r.ServeHTTP(w, req)

				// Converting the query body into a model
				Expect(json.Unmarshal(w.Body.Bytes(), &tokens)).To(BeNil())
			})

			It("should return a couple of new tokens", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(tokens).NotTo(Equal(nil))
			})
		})
	})

	Describe("Update user icon", func() {
		BeforeEach(func() {
			r.PUT("/user/settings/update/icon", handler.UpdateUserIcon)
		})

		Context("no such file", func() {
			BeforeEach(func() {
				// Preparing an icon for upload
				body := new(bytes.Buffer)
				writer := multipart.NewWriter(body)
				Expect(writer.Close()).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/icon", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the icon has not been transferred", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"No such file."}`))
			})
		})

		Context("icon is too large.", func() {
			BeforeEach(func() {
				// Preparing an icon for upload
				body, writer, err := filePreparation("./static/test_big_icon.png")
				Expect(err).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/icon", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the icon is too large", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Icon is too large."}`))
			})
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Preparing an icon for upload
				body, writer, err := filePreparation("./static/test_icon.png")
				Expect(err).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/icon", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("incorrect user icon file type", func() {
			BeforeEach(func() {
				// Preparing an icon for upload
				body, writer, err := filePreparation("./static/test_icon.webp")
				Expect(err).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/icon", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the icon file type is incorrect", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect user icon file type."}`))
			})
		})

		Context("ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserIcon)).
					WithArgs("user-117115101114.png", 117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

				// Preparing an icon for upload
				body, writer, err := filePreparation("./static/test_icon.png")
				Expect(err).To(BeNil())

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPut, "/user/settings/update/icon", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the user icon has been successfully updated", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"Icon successfully updated."}`))
			})
		})
	})

	Describe("Delete user icon", func() {
		BeforeEach(func() {
			r.DELETE("/user/delete/icon", handler.DeleteUserIcon)
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(0).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, "/user/delete/icon", nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", ""))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, "/user/delete/icon", nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user icon is not yet installed", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"The user icon is not yet installed."}`))
			})
		})

		Context("ok", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", "", "user-117115101114.png"))

				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlEditUserIcon)).
					WithArgs("", 117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

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
				req := httptest.NewRequest(http.MethodDelete, `/user/delete/icon`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)

			})

			It("should return a message that the user icon was deleted successfully", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"The user icon has been deleted."}`))
			})
		})
	})

	Describe("Delete user", func() {
		BeforeEach(func() {
			r.DELETE("/user/delete/account", handler.DeleteUser)
		})

		Context("inactive user", func() {
			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(0).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/user/delete/account?password=StRon9Pa$$w0rd`, nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error that the user is inactive", func() {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				Expect(w.Body.String()).To(Equal(`{"error":"Inactive user."}`))
			})
		})

		Context("wrong password", func() {
			BeforeEach(func() {
				// Password hash generation
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte("wErRyStRon9Pa$$w0rd"), bcrypt.DefaultCost)
				Expect(err).To(BeNil())

				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", hashedPassword, "user-117115101114.png"))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodDelete, `/user/delete/account?password=StRon9Pa$$w0rd`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return an error about incorrect password", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect password."}`))
			})
		})

		Context("deleting account with icon, lists, tasks and subtasks", func() {
			BeforeEach(func() {
				// Password hash generation
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte("StRon9Pa$$w0rd"), bcrypt.DefaultCost)
				Expect(err).To(BeNil())

				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserById)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(117115101114, "email@example.com", "Test User Name", "test_username", hashedPassword, "user-117115101114.png"))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllListsByUserId)).
					WithArgs(117115101114).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "comment", "index"}).
						AddRow(108105115116, 117115101114, "Test List Name", "Test List Comment", 0))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllTasksByListId)).
					WithArgs(108105115116).
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

				postgresMock.ExpectBegin()
				postgresMock.ExpectExec(regexp.QuoteMeta(models.SqlDeleteUser)).
					WithArgs(117115101114).
					WillReturnResult(sqlmock.NewResult(1, 1))
				postgresMock.ExpectCommit()

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
				req := httptest.NewRequest(http.MethodDelete, `/user/delete/account?password=StRon9Pa$$w0rd`, nil)
				req.Header.Set("token", accessJwt)
				r.ServeHTTP(w, req)
			})

			It("should return a message that the account was successfully deleted", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"The account has been deleted."}`))
			})
		})
	})
})
