#Http

Пакет позволяет работать с http без создания и конфигурирования кастомных клиентов в удобном виде с сохранением ответа в модели

###Пример
```go

var response struct{
		Ip string
	}
err :=	Get("http://ip.jsontest.com").
		Param("format_id",1).
		Param("anyparam","string value").
	 Do2Json(context.Background(), &response)

if err != nil {
    panic(err)
}
	
fmt.Println("You ip", response.Ip)