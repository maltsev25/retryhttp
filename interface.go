package retryhttp

import (
	"net/http"

	"github.com/mailru/easyjson"
)

type ClientInterfaceExample interface {
	Do(request *http.Request) (*http.Response, error)
	Do2Bytes(request *http.Request) ([]byte, error)
	Do2JSON(request *http.Request, receiver interface{}) error
	Do2EasyJSON(request *http.Request, receiver easyjson.Unmarshaler) error
}
