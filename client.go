package retryhttp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
)

type Config struct {
	TryQty    uint
	WaitRetry time.Duration
	Timeout   time.Duration
	Delay     time.Duration
	MaxJitter time.Duration
	Name      string
}

type Properties struct {
	Config
	OnRetry        retry.OnRetryFunc
	DelayType      retry.DelayTypeFunc
	RetryHTTPCodes []int
}

var (
	defaultClient *Client
	mu            sync.Mutex
	defaultOnce   sync.Once
	defaultConfig = Config{
		TryQty:    3,
		WaitRetry: time.Second * 2,
		Timeout:   time.Second,
		Delay:     time.Millisecond * 100,
		MaxJitter: time.Millisecond * 10,
		Name:      "retry_http",
	}
)

type Client struct {
	client         *http.Client
	retryOptions   []retry.Option
	retryHTTPCodes []int
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

func New(customize ...func(properties *Properties)) *Client {
	prop := &Properties{
		Config:    defaultConfig,
		DelayType: retry.DefaultDelayType,
		OnRetry:   func(n uint, err error) {},
	}
	for _, c := range customize {
		c(prop)
	}
	opts := []retry.Option{
		retry.MaxDelay(prop.WaitRetry),
		retry.Attempts(prop.TryQty),
		retry.OnRetry(prop.OnRetry),
		retry.Delay(prop.Delay),
		retry.MaxJitter(prop.MaxJitter),
		retry.DelayType(prop.DelayType),
		retry.LastErrorOnly(true),
	}
	client := cleanhttp.DefaultPooledClient()
	client.Timeout = prop.Timeout

	return &Client{client: client, retryOptions: opts}
}

func Request(request *http.Request) *Builder { return Default().Request(request) }

func (c *Client) Request(request *http.Request) *Builder {
	return &Builder{client: c, request: request}
}

type Builder struct {
	client  *Client
	request *http.Request
}

// Do2EasyJSON - выполняет http метод и сохраняет ответ в easy json структуру, на которую передан указатель
func (h *Builder) Do2EasyJSON(ctx context.Context, receiver easyjson.Unmarshaler) error {
	bytes, err := h.Do2Bytes(ctx)
	if err != nil {
		return err
	}
	if receiver == nil {
		return nil
	}
	err = easyjson.Unmarshal(bytes, receiver)
	if err != nil {
		return errors.Wrap(err, "easyjson.UnmarshalFromReader")
	}
	return nil
}

// Do2JSON - выполняет http метод и сохраняет json ответ в структуру, на которую передан указатель
func (h *Builder) Do2JSON(ctx context.Context, receiver interface{}) error {
	bytes, err := h.Do2Bytes(ctx)
	if err != nil {
		return err
	}
	if receiver == nil {
		return nil
	}
	err = json.Unmarshal(bytes, receiver)
	if err != nil {
		return errors.Wrap(err, "json.Unmarshal")
	}
	return nil
}

// Do2Bytes - выполняет http метод и выдает результат в []byte
func (h *Builder) Do2Bytes(ctx context.Context) ([]byte, error) {
	response, err := h.client.doWithRetries(ctx, h.request)
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
		return nil, err
	}
	return result, nil
}

// Do - выполняет http метод и выдает результат в *http.Response
func (h *Builder) Do(ctx context.Context) (*http.Response, error) {
	return h.client.doWithRetries(ctx, h.request)
}

func (c *Client) doWithRetries(ctx context.Context, request *http.Request) (resp *http.Response, err error) {
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
			for _, httpCode := range c.retryHTTPCodes {
				if resp.StatusCode == httpCode {
					return errors.Errorf("get code %d", resp.StatusCode)
				}
			}
		}

		return nil
	}

	opts := make([]retry.Option, 0, len(c.retryOptions)+1)
	opts = append(opts, retry.Context(ctx))
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
	if _, ok := err.(net.Error); ok {
		return true
	}
	return false
}
