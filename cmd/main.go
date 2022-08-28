package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/NKTKLN/todo-api/pkg/common"
	"github.com/NKTKLN/todo-api/pkg/db/minio"
	"github.com/NKTKLN/todo-api/pkg/db/postgres"
	"github.com/NKTKLN/todo-api/pkg/db/redis"
	"github.com/NKTKLN/todo-api/pkg/handlers"
	"github.com/NKTKLN/todo-api/server"
)

// @title        ToDo API
// @version      1.0
// @description  some description

// @contact.name   API Support
// @contact.url    https://nktkln.com
// @contact.email  nktkln@nktkln.com

// @securityDefinitions.apikey  token
// @in                          header
// @name                        token

// @license.name  MIT
// @license.url   https://github.com/NKTKLN/todo-api/blob/main/LICENSE

// @BasePath  /
func main() {
	if err := initConfig(); err != nil {
		logrus.Fatalf("Error initializing configs: %s", err.Error())
	}

	postgresDB, err := postgres.Connect(
		viper.GetString("databases.postgres.host"),
		viper.GetString("databases.postgres.user"), 
		viper.GetString("databases.postgres.password"), 
		viper.GetString("databases.postgres.db-name"),
		viper.GetInt("databases.postgres.port"),
	)
	if err != nil {
		logrus.Fatalf("error when connecting to the postgres database: %s", err.Error())
	}

	redisClient, err := redis.Connect(
		fmt.Sprintf("%s:%d", viper.GetString("databases.redis.host"), viper.GetInt("databases.redis.port")),
		viper.GetString("databases.redis.password"),
		viper.GetInt("databases.redis.email-db"),
		viper.GetInt("databases.redis.access-token-db"),
		viper.GetInt("databases.redis.refresh-token-db"),
	)
	if err != nil {
		logrus.Fatalf("error when connecting to the redis database: %s", err.Error())
	}

	minioClient := minio.NewMinioProvider(
		fmt.Sprintf("%s:%d", viper.GetString("databases.minio.host"), viper.GetInt("databases.minio.port")),
		viper.GetString("databases.minio.user"), 
		viper.GetString("databases.minio.password"), 
		viper.GetBool("databases.minio.ssl"),
	)
	if err = minioClient.Connect(); err != nil {
		logrus.Fatalf("error when connecting to the MinIO database: %s", err.Error())
	}
	if err = minioClient.CreateBucket(context.Background()); err != nil {
		logrus.Fatalf("error when creting bucket in MinIO database: %s", err.Error())
	}

	emailAuthData := common.NewEmailProvider(
		viper.GetString("smtp.email"), 
		viper.GetString("smtp.password"), 
		viper.GetString("smtp.server"),
		viper.GetInt("smtp.port"),
	)

	handler := handlers.Handler{
		PostgresDB:    postgresDB,
		RedisClient:   redisClient,
		MinIOClient:   minioClient,
		EmailAuthData: emailAuthData,
	}

	srv := new(server.Server)
	go func() {
		if err := srv.Run(viper.GetString("api.port"), handler.InitRoutes()); err != nil {
			logrus.Fatalf("error occured while running http server: %s", err.Error())
		}
	}()

	logrus.Print("TodoApi Started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Print("TodoApi Shutting Down")

	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("error occured on server shutting down: %s", err.Error())
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
