package retryhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDefaultConfig tests the default config
func TestGetDefaultConfig(t *testing.T) {
	assert.Equal(t, GetDefaultConfig(), Config{
		Attempts:  3,
		MaxDelay:  2 * time.Second,
		Timeout:   1 * time.Second,
		Delay:     100 * time.Millisecond,
		MaxJitter: 10 * time.Millisecond,
	})
}

// TestSetDefaultConfig tests that the default config is set correctly
func TestSetDefaultConfig(t *testing.T) {
	config := Config{
		Attempts:  1,
		MaxDelay:  2 * time.Second,
		Timeout:   3 * time.Second,
		Delay:     100 * time.Millisecond,
		MaxJitter: 10 * time.Millisecond,
	}
	SetDefaultConfig(config)
	assert.Equal(t, GetDefaultConfig(), config)
}

type testData struct {
	A int
	B float64
}

type testCase struct {
	name                 string
	url                  string
	method               string
	expectedStatusCode   int
	expectedByteResponse []byte
	expectedResponse     testData
}

// TestGet tests the Get method
func TestGet(t *testing.T) {
	server := getServer()
	defer server.Close()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		if tc.method != http.MethodGet {
			continue
		}

		t.Run(tc.name, func(t *testing.T) {
			res, err := Get(tc.url)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestPost tests the Post method
func TestPost(t *testing.T) {
	server := getServer()
	defer server.Close()
	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		if tc.method != http.MethodPost {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			res, err := Post(tc.url, "application/json", nil)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestPostForm tests the PostForm method
func TestPostForm(t *testing.T) {
	server := getServer()
	defer server.Close()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		if tc.method != http.MethodPost {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			res, err := PostForm(tc.url, nil)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestHead tests the Head method
func TestHead(t *testing.T) {
	server := getServer()
	defer server.Close()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Head(tc.url)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestDo tests the Do method
func TestDo(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := Do(request)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestDo2Bytes tests the Do2Bytes method
func TestDo2Bytes(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := Do2Bytes(request)
			require.NoError(t, err)
			require.Equal(t, res, tc.expectedByteResponse)
		})
	}
}

// TestDo2JSON tests the Do2JSON method
func TestDo2JSON(t *testing.T) {
	server := getServer()
	defer server.Close()

	var d testData
	ctx := context.Background()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			err = Do2JSON(request, &d)
			require.Equal(t, d, tc.expectedResponse)
		})
	}
}

// TestCustomClientDo2Bytes tests the Do2Bytes method with custom client
func TestCustomClientDo2Bytes(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.Attempts = 1
		properties.Timeout = time.Millisecond * 2
	}
	client := New(prop)

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := client.Do2Bytes(request)
			require.NoError(t, err)
			require.Equal(t, res, tc.expectedByteResponse)
		})
	}
}

// TestCustomClientDoRetry tests the DoRetry method with custom client
func TestCustomClientDoRetry(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.Attempts = 3
		properties.Timeout = time.Millisecond * 3
	}
	client := New(prop)

	testCases := []testCase{
		{
			name:               "success after retry method " + http.MethodGet,
			url:                fmt.Sprintf("%s/long", server.URL),
			method:             http.MethodGet,
			expectedStatusCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := client.Do(request)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestDo2JSONNoReceiver tests the Do2JSON method with no receiver
func TestDo2JSONNoReceiver(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := []struct {
		name          string
		url           string
		method        string
		expectedError error
	}{
		{
			name:          "receiver not set " + http.MethodGet,
			url:           fmt.Sprintf("%s/url", server.URL),
			method:        http.MethodGet,
			expectedError: ErrReceiverNotSet,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			err = Do2JSON(request, nil)
			require.Equal(t, err, tc.expectedError)
		})
	}
}

// TestCustomClientDoRetryStatusCode tests the DoRetry method with custom client and StatusCode
func TestCustomClientDoRetryStatusCode(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.RetryHTTPCodes = []int{500, 400}
		properties.Attempts = 4
	}
	client := New(prop)

	testCases := []testCase{
		{
			name:               "success after retry method " + http.MethodGet,
			url:                fmt.Sprintf("%s/different_response", server.URL),
			method:             http.MethodGet,
			expectedStatusCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := client.Do(request)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestCustomClientDoRetryLogicFunc tests the DoRetry method with custom client and RetryLogicFunc
func TestCustomClientDoRetryLogicFunc(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.RetryLogicFunc = func(r *http.Response) error {
			if r.StatusCode == http.StatusInternalServerError {
				return errors.New("error")
			}
			return nil
		}
		properties.Attempts = 3
	}
	client := New(prop)

	testCases := []testCase{
		{
			name:               "success after retry method " + http.MethodPost,
			url:                fmt.Sprintf("%s/different_response", server.URL),
			method:             http.MethodPost,
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	test := testData{A: 1, B: 2.0}
	b, _ := json.Marshal(test)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, bytes.NewBuffer(b))
			require.NoError(t, err)

			res, err := client.Do(request)
			require.NoError(t, err)
			defer func() {
				if res != nil {
					_ = res.Body.Close()
				}
			}()

			require.Equal(t, res.StatusCode, tc.expectedStatusCode)
		})
	}
}

// TestUnrecoverableError tests the UnrecoverableError
func TestUnrecoverableError(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := []testCase{
		{
			name:   "unrecoverable error " + http.MethodGet,
			url:    "not a valid url",
			method: http.MethodGet,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			_, err = Do(request)
			require.Error(t, err)
		})
	}
}

// TestIsRetryableError tests the IsRetryableError method
func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		result bool
	}{
		{
			name:   "no error",
			err:    nil,
			result: false,
		},
		{
			name: "retryable net error",
			err: &net.DNSError{
				Err:       "context deadline exceeded (Client.Timeout exceeded while awaiting headers)",
				IsTimeout: true,
			},
			result: true,
		},
		{
			name:   "not retryable http error",
			err:    http.ErrServerClosed,
			result: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, isRetryableError(tc.err), tc.result)
		})
	}
}

func getTestCases(url string) []testCase {
	flt := math.Floor(rand.Float64()*10000) / 10000
	data := testData{rand.Int(), flt}
	bytes, _ := json.Marshal(data)

	testCases := []testCase{
		{
			name:                 "success method " + http.MethodGet,
			url:                  fmt.Sprintf("%s/url?a=%d&b=%f", url, data.A, data.B),
			method:               http.MethodGet,
			expectedStatusCode:   http.StatusOK,
			expectedByteResponse: bytes,
			expectedResponse:     data,
		},
		{
			name:                 "success method " + http.MethodPost,
			url:                  fmt.Sprintf("%s/url?a=%d&b=%f", url, data.A, data.B),
			method:               http.MethodPost,
			expectedStatusCode:   http.StatusOK,
			expectedByteResponse: bytes,
			expectedResponse:     data,
		},
		{
			name:                 "success method " + http.MethodPut,
			url:                  fmt.Sprintf("%s/url?a=%d&b=%f", url, data.A, data.B),
			method:               http.MethodPut,
			expectedStatusCode:   http.StatusOK,
			expectedByteResponse: bytes,
			expectedResponse:     data,
		},
		{
			name:                 "success method " + http.MethodDelete,
			url:                  fmt.Sprintf("%s/url?a=%d&b=%f", url, data.A, data.B),
			method:               http.MethodDelete,
			expectedStatusCode:   http.StatusOK,
			expectedByteResponse: bytes,
			expectedResponse:     data,
		},
		{
			name:                 "success method " + http.MethodPatch,
			url:                  fmt.Sprintf("%s/url?a=%d&b=%f", url, data.A, data.B),
			method:               http.MethodPatch,
			expectedStatusCode:   http.StatusOK,
			expectedByteResponse: bytes,
			expectedResponse:     data,
		},
		{
			name:                 "400 method " + http.MethodGet,
			url:                  fmt.Sprintf("%s/400", url),
			method:               http.MethodGet,
			expectedStatusCode:   http.StatusBadRequest,
			expectedByteResponse: []byte{},
			expectedResponse:     data,
		},
		{
			name:                 "400 method " + http.MethodPost,
			url:                  fmt.Sprintf("%s/400", url),
			method:               http.MethodPost,
			expectedStatusCode:   http.StatusBadRequest,
			expectedByteResponse: []byte{},
			expectedResponse:     data,
		},
		{
			name:                 "500 method " + http.MethodGet,
			url:                  fmt.Sprintf("%s/500", url),
			method:               http.MethodGet,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedByteResponse: []byte{},
			expectedResponse:     data,
		},
		{
			name:                 "500 method " + http.MethodPost,
			url:                  fmt.Sprintf("%s/500", url),
			method:               http.MethodPost,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedByteResponse: []byte{},
			expectedResponse:     data,
		},
	}
	return testCases
}

func getServer() *httptest.Server {
	var (
		counter  time.Duration
		switcher int
	)
	counter = 2
	mux := http.NewServeMux()
	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		a, _ := strconv.Atoi(query.Get("a"))
		b, _ := strconv.ParseFloat(query.Get("b"), 64)
		d1 := testData{A: a, B: b}
		buf, err := json.Marshal(d1)
		if err != nil {
			panic(err)
		}
		_, err = w.Write(buf)
		if err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("/long", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * counter)
		counter--
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/different_response", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get("Content-Length") != "0" {
			len, _ := strconv.Atoi(r.Header.Get("Content-Length"))
			buf := make([]byte, len)
			read, _ := r.Body.Read(buf)
			if read != len {
				panic("read != len")
			}
		}
		var httpStatus int
		switch switcher {
		case 0, 1:
			httpStatus = http.StatusInternalServerError
		case 2:
			httpStatus = http.StatusBadRequest
		case 3:
			httpStatus = http.StatusOK
		}
		switcher++
		w.WriteHeader(httpStatus)
	})
	mux.HandleFunc("/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	return httptest.NewServer(mux)
}
