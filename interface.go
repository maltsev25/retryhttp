package retryhttp

import (
	"io"
	"net/http"
	"net/url"

	"github.com/mailru/easyjson"
)

type ClientHTTP interface {
	Get(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	PostForm(url string, data url.Values) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	Do(request *http.Request) (*http.Response, error)
	Do2Bytes(request *http.Request) ([]byte, error)
	Do2JSON(request *http.Request, receiver interface{}) error
	Do2EasyJSON(request *http.Request, receiver easyjson.Unmarshaler) error
}
