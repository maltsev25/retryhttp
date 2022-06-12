#retryhttp

###Example
```go

var response struct{
	Ip string
}
request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip.jsontest.com", nil)
err = Request(request).Do2JSON(ctx, &response)
if err != nil {
    panic(err)
}
	
fmt.Println("You ip", response.Ip)