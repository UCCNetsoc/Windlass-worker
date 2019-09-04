package host

import (
	"net/http"
)

type Error struct {
	StatusCode int
	message    string
}

func newError(message string, status int) Error {
	return Error{status, message}
}

func (e Error) Error() string {
	return e.message
}

var (
	ErrHostExists error = newError("container host aleady exists", http.StatusConflict)
)
