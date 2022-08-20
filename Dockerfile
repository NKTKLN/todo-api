FROM golang:alpine as builder

# Copying the project into a build container
COPY go.mod go.sum /go/src/github.com/NKTKLN/todo-api/
WORKDIR /go/src/github.com/NKTKLN/todo-api

RUN go mod download
COPY . /go/src/github.com/NKTKLN/todo-api

# Building an application bin file
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app github.com/NKTKLN/todo-api/cmd

# Packing the assembled application into a smaller image
FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

# Copying the assembled project into the main container
COPY --from=builder /go/src/github.com/NKTKLN/todo-api/ /usr/bin/todo-api/
WORKDIR /usr/bin/todo-api/

EXPOSE 80

CMD ["./app"]
