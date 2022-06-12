package retryhttp

import (
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultConfig(t *testing.T) {
	assert.Equal(t, GetDefaultConfig(), Config{
		TryQty:    3,
		WaitRetry: 2 * time.Second,
		Timeout:   1 * time.Second,
		Delay:     100 * time.Millisecond,
		MaxJitter: 10 * time.Millisecond,
		Name:      "retry_http"})
}

func TestSetDefaultConfig(t *testing.T) {
	config := Config{
		TryQty:    2,
		WaitRetry: 2 * time.Second,
		Timeout:   3 * time.Second,
		Delay:     100 * time.Millisecond,
		MaxJitter: 10 * time.Millisecond,
		Name:      "retry_http"}
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

func TestDo(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := Request(request).Do(ctx)
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

func TestDo2Bytes(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := Request(request).Do2Bytes(ctx)
			require.NoError(t, err)
			require.Equal(t, res, tc.expectedByteResponse)
		})
	}
}

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

			err = Request(request).Do2JSON(ctx, &d)
			require.Equal(t, d, tc.expectedResponse)
		})
	}
}

func TestCustomClientDo2Bytes(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.Name = "custom_retry_http"
		properties.TryQty = 1
		properties.Timeout = time.Millisecond * 2
	}
	client := New(prop)

	testCases := getTestCases(server.URL)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
			require.NoError(t, err)

			res, err := client.Request(request).Do2Bytes(ctx)
			require.NoError(t, err)
			require.Equal(t, res, tc.expectedByteResponse)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		result bool
	}{
		{
			name:   "not retryable",
			err:    nil,
			result: false,
		},
		{
			name: "retryable",
			err: &net.DNSError{
				Err:       "context deadline exceeded (Client.Timeout exceeded while awaiting headers)",
				IsTimeout: true,
			},
			result: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, isRetryableError(tc.err), tc.result)
		})
	}
}

func TestCustomClientDoRetry(t *testing.T) {
	server := getServer()
	defer server.Close()

	ctx := context.Background()

	prop := func(properties *Properties) {
		properties.TryQty = 3
		properties.Timeout = time.Millisecond * 2
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

			res, err := client.Request(request).Do(ctx)
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
			name:                 "500 method " + http.MethodGet,
			url:                  fmt.Sprintf("%s/500", url),
			method:               http.MethodGet,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedByteResponse: []byte{},
			expectedResponse:     data,
		},
	}
	return testCases
}

func getServer() *httptest.Server {
	var counter time.Duration
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
	mux.HandleFunc("/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	return httptest.NewServer(mux)
}
