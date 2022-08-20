package models

type ApiError struct {
	Error string `json:"error"`
}

type ApiMessage struct {
	Message string `json:"message"`
}
