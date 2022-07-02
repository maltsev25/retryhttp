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