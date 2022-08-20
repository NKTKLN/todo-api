package tests

import (
	"context"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/db"
	pg "github.com/NKTKLN/todo-api/pkg/db/postgres"
)

// Test db initialization
func MockPostgresConnection() (db.PostgresDB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		logrus.Fatalln(err)
	}

	DB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		logrus.Fatalln(err)
	}

	return &pg.PDB{DB: DB}, mock
}

func TestRedisConnection() *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		logrus.Fatalln(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client
}

// Fake email provider
type fakeEmailAuthData struct {
	email    string
	password string
	server   string
	port     int
}

type fakeEmailProvider interface {
	UserEmailVerification(context.Context, db.RedisClient, models.UserData) error
	UserPasswordReset(context.Context, db.RedisClient, string) error
	UserEmailReset(context.Context, db.RedisClient, string, int) error
}

func NewFakeEmailProvider(senderEmail, emailPassword, emailServer string, emailServerPort int) fakeEmailProvider {
	return &fakeEmailAuthData{
		email:    senderEmail,
		password: emailPassword,
		server:   emailServer,
		port:     emailServerPort,
	}
}

func (d *fakeEmailAuthData) UserEmailVerification(ctx context.Context, client db.RedisClient, data models.UserData) (err error) {
	return
}

func (d *fakeEmailAuthData) UserPasswordReset(ctx context.Context, client db.RedisClient, userEmail string) (err error) {
	return
}

func (d *fakeEmailAuthData) UserEmailReset(ctx context.Context, client db.RedisClient, userEmail string, userId int) (err error) {
	return
}
