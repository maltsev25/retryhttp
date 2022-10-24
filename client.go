package retryhttp

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
)

type Config struct {
	// Attempts must be greater than 0, otherwise it will be set to 1
	Attempts  uint
	MaxDelay  time.Duration
	Timeout   time.Duration
	Delay     time.Duration
	MaxJitter time.Duration
}

type Properties struct {
	Config
	OnRetry        retry.OnRetryFunc
	DelayType      retry.DelayTypeFunc
	RetryLogicFunc RetryLogicFunc
	RetryHTTPCodes []int
}

// RetryLogicFunc is a function that returns error if request should be retried
// disabling check for RetryHTTPCodes
type RetryLogicFunc func(resp *http.Response) error

var (
	defaultClient *Client
	mu            sync.Mutex
	defaultOnce   sync.Once
	defaultConfig = Config{
		Attempts:  3,
		MaxDelay:  time.Second * 2,
		Timeout:   time.Second,
		Delay:     time.Millisecond * 100,
		MaxJitter: time.Millisecond * 10,
	}
	ErrReceiverNotSet = errors.New("receiver not set")
)

type Client struct {
	client         *http.Client
	retryOptions   []retry.Option
	retryLogicFunc RetryLogicFunc
}

func SetDefaultConfig(config Config) {
	mu.Lock()
	defer mu.Unlock()
	if defaultClient != nil {
		panic("can not retry_http.SetDefaultConfig after call retry_http.Default()")
	}
	defaultConfig = config
}

func GetDefaultConfig() Config {
	mu.Lock()
	defer mu.Unlock()
	return defaultConfig
}

func Default() *Client {
	defaultOnce.Do(func() {
		defaultClient = New()
	})
	return defaultClient
}

// Get execute http method and return result into *http.Response
func Get(url string) (*http.Response, error) {
	return Default().Get(url)
}

// Post execute http method and return result into *http.Response
func Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return Default().Post(url, contentType, body)
}

// PostForm execute http method and return result into *http.Response
func PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return Default().PostForm(url, data)
}

// Head execute http method and return result into *http.Response
func Head(url string) (resp *http.Response, err error) {
	return Default().Head(url)
}

// Do execute http method and return result into *http.Response
func Do(request *http.Request) (*http.Response, error) {
	return Default().Do(request)
}

// Do2Bytes execute http method and return result into []byte
func Do2Bytes(request *http.Request) ([]byte, error) {
	return Default().Do2Bytes(request)
}

// Do2JSON execute http method and save json response into receiver struct
func Do2JSON(request *http.Request, receiver interface{}) error {
	return Default().Do2JSON(request, receiver)
}

// Do2EasyJSON execute http method and save response into easyjson receiver struct
func Do2EasyJSON(request *http.Request, receiver easyjson.Unmarshaler) error {
	return Default().Do2EasyJSON(request, receiver)
}

func New(customize ...func(properties *Properties)) *Client {
	prop := &Properties{
		Config:    defaultConfig,
		DelayType: retry.DefaultDelayType,
		OnRetry:   func(n uint, err error) {},
	}
	for _, c := range customize {
		c(prop)
	}
	if prop.RetryLogicFunc == nil {
		prop.RetryLogicFunc = func(resp *http.Response) error {
			for _, httpCode := range prop.RetryHTTPCodes {
				if resp.StatusCode == httpCode {
					return errors.Errorf("got code %d", resp.StatusCode)
				}
			}
			return nil
		}
	}
	if prop.Attempts < 1 {
		prop.Attempts = 1
	}

	opts := []retry.Option{
		retry.MaxDelay(prop.MaxDelay),
		retry.Attempts(prop.Attempts),
		retry.OnRetry(prop.OnRetry),
		retry.Delay(prop.Delay),
		retry.MaxJitter(prop.MaxJitter),
		retry.DelayType(prop.DelayType),
		retry.LastErrorOnly(true),
	}
	client := cleanhttp.DefaultPooledClient()
	client.Timeout = prop.Timeout

	return &Client{client: client, retryOptions: opts, retryLogicFunc: prop.RetryLogicFunc}
}

// Get execute http method and return result into *http.Response
func (c *Client) Get(url string) (resp *http.Response, err error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.doWithRetries(request)
}

// Post execute http method and return result into *http.Response
func (c *Client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", contentType)
	return c.doWithRetries(request)
}

// PostForm execute http method and return result into *http.Response
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Head execute http method and return result into *http.Response
func (c *Client) Head(url string) (resp *http.Response, err error) {
	request, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.doWithRetries(request)
}

// Do execute http method and return result into *http.Response
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	return c.doWithRetries(request)
}

// Do2Bytes execute http method and return result into []byte
func (c *Client) Do2Bytes(request *http.Request) ([]byte, error) {
	response, err := c.doWithRetries(request)
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadAll")
	}
	return result, nil
}

// Do2JSON execute http method and save json response into receiver struct
func (c *Client) Do2JSON(request *http.Request, receiver interface{}) error {
	bytes, err := c.Do2Bytes(request)
	if err != nil {
		return err
	}
	if receiver == nil {
		return ErrReceiverNotSet
	}
	err = json.Unmarshal(bytes, receiver)
	if err != nil {
		return errors.Wrap(err, "json.Unmarshal")
	}
	return nil
}

// Do2EasyJSON execute http method and save response into easyjson receiver struct
func (c *Client) Do2EasyJSON(request *http.Request, receiver easyjson.Unmarshaler) error {
	bytes, err := c.Do2Bytes(request)
	if err != nil {
		return err
	}
	if receiver == nil {
		return ErrReceiverNotSet
	}
	err = easyjson.Unmarshal(bytes, receiver)
	if err != nil {
		return errors.Wrap(err, "easyjson.Unmarshal")
	}
	return nil
}

func (c *Client) doWithRetries(request *http.Request) (resp *http.Response, err error) {
	action := func() error {
		//nolint:bodyclose
		resp, err = c.client.Do(request)
		if err != nil {
			if isRetryableError(err) {
				return err
			}
			return retry.Unrecoverable(err)
		}

		if resp != nil {
			return c.retryLogicFunc(resp)
		}

		return nil
	}

	opts := make([]retry.Option, 0, len(c.retryOptions)+1)
	opts = append(opts, retry.Context(request.Context()))
	opts = append(opts, c.retryOptions...)

	err = retry.Do(action, opts...)
	if err != nil {
		return
	}
	return resp, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errNet, ok := err.(net.Error)
	if ok && errNet.Timeout() {
		return true
	}
	return false
}
