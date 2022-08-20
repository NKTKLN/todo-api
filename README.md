# 📋 ToDo API

## 💾 Prerequisites

* [Golang](https://go.dev/)
* [Docker CE](https://docs.docker.com/engine/install/)
* [Docker Compose](https://docs.docker.com/compose/install/)

## ⚙️ Running api

* 🐳 Run in Docker

  ```
  make docker-network
  make docker-build
  ```
* 📄 Build to binary file

  > If you do not want to run the application via docker, then run this command to create a binary with the program (to run the binary you need to install and configure: PostgreSQL, Redis, MinIO)

  ```
  make build
  ```
* 💻 Local run

  > If you want to test the application on the local machine without using docker, run this command to start it (to start it you need to install and configure: PostgreSQL, Redis, MinIO)

  ```
  make run
  ```
* 🧪 Test run

  > If you want to run application testing, run this command to start it (to run it you need to install and configure: PostgreSQL, Redis, MinIO)

  ```
  make test
  ```

## 📃 License

### All my apps are released under the MIT license, see [LICENSE.md](https://github.com/NKTKLN/todo-api/blob/master/LICENSE) for full text.
