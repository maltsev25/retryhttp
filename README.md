retryhttp
===========

The `retryhttp` package is HTTP client with configurable retry policy.

Example Use
===========
The `retryhttp` package is identical to what you would do with
`net/http`. Example:
```go
resp, err := retryhttp.Get("/foo")
if err != nil {
    panic(err)
}
```

Also was added additional methods Do2Bytes, Do2JSON and Do2EasyJSON. Example:
```go
var response struct{
	IP string
}
request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip.jsontest.com", nil)
err = retryhttp.Do2JSON(request, &response)
if err != nil {
    panic(err)
}
	
fmt.Println("You ip", response.IP)
```

Override retry policy
===========
User can override retry policy for client:
```go
prop := func(properties *Properties) {
    properties.Attempts = 10
    properties.Timeout = time.Second
}
client := New(prop)
```

Override JSON Unmarshal
===========
User can register choice of JSON library into `retryhttp` or write own. By default `retryhttp` registers standard `encoding/json` respectively.
```go
prop := func(properties *Properties) {
    properties.JSONUnmarshal = json.Unmarshal
}
client := New(prop)
```