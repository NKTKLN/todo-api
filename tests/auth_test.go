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
	"golang.org/x/crypto/bcrypt"

	"github.com/NKTKLN/todo-api/models"
	rd "github.com/NKTKLN/todo-api/pkg/db/redis"
	"github.com/NKTKLN/todo-api/pkg/handlers"
)

var _ = Describe("Auth", func() {
	var (
		r                       *gin.Engine
		w                       *httptest.ResponseRecorder
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
	})

	AfterEach(func() {
		redisClientEmail.Close()
		redisClientAccessToken.Close()
		redisClientRefreshToken.Close()

		Expect(postgresMock.ExpectationsWereMet()).ShouldNot(HaveOccurred())
	})

	Describe("Sign Up", func() {
		BeforeEach(func() {
			r.POST("/auth/sign-up", handler.SignUp)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", nil)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Context("incorrect data", func() {
			const requestBody = `{"email": "", "name": "", "password": "", "username": ""}`

			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return an error that the input data is invalid", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Incorrect data."}`))
			})
		})

		Describe("Checking for the presence of data in the db", func() {
			const requestBody = `{"email": "email@example.com", "name": "Test Name", "password": "StRon9Pa$$w0rd", "username": "test_username"}`

			Context("mail is already in use", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
							AddRow(117115101114, "email@example.com", "", "test_username", "", ""))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error stating that this mail is already in use", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Mail is already in use."}`))
				})
			})

			Context("username is already in use", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
						WithArgs("test_username").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
							AddRow(117115101114, "", "", "test_username", "", ""))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error stating that this username is already in use", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Username is already in use."}`))
				})
			})
		})

		Describe("Check for correct user data", func() {
			Context("incorrect email", func() {
				BeforeEach(func() {
					const requestBody = `{"email": "test@nktkln", "name": "Test Name", "password": "StRon9Pa$$w0rd", "username": "test_username"}`

					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("test@nktkln").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
						WithArgs("test_username").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error that the mail is invalid", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Incorrect email."}`))
				})
			})

			Context("incorrect name", func() {
				BeforeEach(func() {
					const requestBody = `{"email": "email@example.com", "name": "Test  Name", "password": "StRon9Pa$$w0rd", "username": "test_username"}`

					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
						WithArgs("test_username").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error that the name is invalid", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Incorrect name."}`))
				})
			})

			Context("incorrect username", func() {
				BeforeEach(func() {
					const requestBody = `{"email": "email@example.com", "name": "Test Name", "password": "StRon9Pa$$w0rd", "username": "test__username"}`

					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
						WithArgs("test__username").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error that the username is invalid", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Incorrect username."}`))
				})
			})
		})

		Context("ok", func() {
			BeforeEach(func() {
				const requestBody = `{"email": "email@example.com", "name": "Test Name", "password": "StRon9Pa$$w0rd", "username": "test_username"}`

				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
					WithArgs("test_username").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				// Sending a query with data
				req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewBufferString(requestBody))
				r.ServeHTTP(w, req)
			})

			It("should return a message that the account confirmation link has been sent", func() {
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(Equal(`{"message":"A verification key was sent to your email."}`))
			})
		})
	})

	Describe("Verify Sign Up", func() {
		BeforeEach(func() {
			r.GET("/auth/verify", handler.VerifySignUp)
		})

		Context("time has expired, your key is not valid", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, "/auth/verify?key=key", nil)
				r.ServeHTTP(w, req)
			})

			It("should return an error about invalid key", func() {
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(Equal(`{"error":"Time has expired, your key is not valid."}`))
			})
		})

		Describe("Checking for data in the db", func() {
			BeforeEach(func() {
				// Сonvert data to json
				jsonData, err := json.Marshal(models.UserData{Email: "email@example.com", Username: "test_username"})
				Expect(err).To(BeNil())

				// Adding data to redis
				redisClientEmail.Set(context.Background(), "key", jsonData, time.Minute)
			})

			Context("mail is already in use", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
							AddRow(117115101114, "email@example.com", "", "test_username", "", ""))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/verify?key=key", nil)
					r.ServeHTTP(w, req)
				})

				It("should return an error that the mail is already in use", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Mail is already in use."}`))
				})
			})

			Context("username is already in use", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
						WithArgs("test_username").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
							AddRow(117115101114, "", "", "test_username", "", ""))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/verify?key=key", nil)
					r.ServeHTTP(w, req)
				})

				It("should return an error that the username is already in use", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Username is already in use."}`))
				})
			})
		})

		Context("Ok", func() {
			var tokens models.UserTokens

			BeforeEach(func() {
				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersByUsername)).
					WithArgs("test_username").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectAllUsersById)).
					WithArgs(AnyInt{}).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

				postgresMock.ExpectBegin()
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlInsertUserData)).
					WithArgs("email@example.com", AnyString{}, "Test Name", "test_username", "", AnyInt{}).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow(0))
				postgresMock.ExpectCommit()

				// Convert data to json
				jsonData, err := json.Marshal(models.UserData{Email: "email@example.com", Username: "test_username", Password: "StRon9Pa$$w0rd", Name: "Test Name"})
				Expect(err).To(BeNil())

				// Adding data to redis
				redisClientEmail.Set(context.Background(), "key", jsonData, time.Minute)

				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, "/auth/verify?key=key", nil)
				r.ServeHTTP(w, req)

				// Converting the query body into a model
				Expect(json.Unmarshal(w.Body.Bytes(), &tokens)).To(BeNil())
			})

			It("should return a couple of new tokens", func() {
				Expect(redisClientEmail.Get(context.Background(), "key").Val()).To(Equal(""))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(tokens).NotTo(Equal(nil))
			})
		})
	})

	Describe("Sign In", func() {
		const requestBody = `{"email": "email@example.com", "password": "StRon9Pa$$w0rd"}`

		BeforeEach(func() {
			r.GET("/auth/sign-in", handler.SignIn)
		})

		Context("data retrieval error", func() {
			BeforeEach(func() {
				// Sending a query with data
				req := httptest.NewRequest(http.MethodGet, "/auth/sign-in", nil)
				r.ServeHTTP(w, req)
			})

			It("should return a data conversion error message", func() {
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(Equal(`{"error":"Data retrieval error."}`))
			})
		})

		Describe("Wrong login", func() {
			Context("wrong email", func() {
				BeforeEach(func() {
					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/sign-in", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error about invalid login data", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Wrong email or password."}`))
				})
			})

			Context("wrong password", func() {
				BeforeEach(func() {
					// Password hash generation
					hashedPassword, err := bcrypt.GenerateFromPassword([]byte("wErRyStRon9Pa$$w0rd"), bcrypt.DefaultCost)
					Expect(err).To(BeNil())

					// Query building for the postgres
					postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserByEmail)).
						WithArgs("email@example.com").
						WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
							AddRow(0, "email@example.com", "", "", hashedPassword, ""))

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/sign-in", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)
				})

				It("should return an error about invalid login data", func() {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(Equal(`{"error":"Wrong email or password."}`))
				})
			})
		})

		Describe("Ok", func() {
			var tokens models.UserTokens

			BeforeEach(func() {
				// Password hash generation
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte("StRon9Pa$$w0rd"), bcrypt.DefaultCost)
				Expect(err).To(BeNil())

				// Query building for the postgres
				postgresMock.ExpectQuery(regexp.QuoteMeta(models.SqlSelectUserByEmail)).
					WithArgs("email@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "username", "password", "icon"}).
						AddRow(0, "email@example.com", "", "", hashedPassword, ""))
			})

			Context("without tokens in the db", func() {
				BeforeEach(func() {
					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/sign-in", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &tokens)).To(BeNil())
				})

				It("should return a couple of new tokens", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(tokens).NotTo(Equal(nil))
				})
			})

			Context("with tokens in the db", func() {
				BeforeEach(func() {
					// Adding data to redis
					ctx := context.Background()
					redisClientRefreshToken.Set(ctx, "0", "VeRy$eCrEt@nDc0mPlExReFrE$Ht0kEn", time.Minute)
					redisClientAccessToken.Set(ctx, "0", "VeRy$eCrEt@nDc0mPlEx@cCe$$T0KeN", time.Minute)

					// Sending a query with data
					req := httptest.NewRequest(http.MethodGet, "/auth/sign-in", bytes.NewBufferString(requestBody))
					r.ServeHTTP(w, req)

					// Converting the query body into a model
					Expect(json.Unmarshal(w.Body.Bytes(), &tokens)).To(BeNil())
				})

				It("should return a couple of tokens from the db", func() {
					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(tokens).To(Equal(models.UserTokens{
						AccessToken:  "VeRy$eCrEt@nDc0mPlEx@cCe$$T0KeN",
						RefreshToken: "VeRy$eCrEt@nDc0mPlExReFrE$Ht0kEn"},
					))
				})
			})
		})
	})
})
