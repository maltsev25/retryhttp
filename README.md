retryhttp
================

example
================

```go
var response struct{
	Ip string
}
request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip.jsontest.com", nil)
err = Do2JSON(request, &response)
if err != nil {
    panic(err)
}
	
fmt.Println("You ip", response.Ip)