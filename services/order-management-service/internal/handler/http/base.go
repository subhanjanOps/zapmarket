package http

import (
	"encoding/json"
	"net/http"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/pkg/httpx"
)

// Response is the standard API envelope — used in swagger annotations only.
type Response struct {
	Success bool        `json:"success"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	httpx.JSON(w, statusCode, data)
}

func SuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	httpx.Success(w, statusCode, data)
}

func ErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	httpx.Error(w, statusCode, code, message)
}

func HandleError(w http.ResponseWriter, err error) {
	pkgerrors.HandleHTTP(w, err)
}

func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
