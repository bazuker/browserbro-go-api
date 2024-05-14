# BrowserBro Golang API Client

Golang client for [BrowserBro](https://github.com/bazuker/browserbro) API.

```bash
go get -u github.com/bazuker/browserbro-go-api
```

Example
```go
package main

import (
    "fmt"
    "github.com/bazuker/browserbro-go-api/client"
)

func main() {
    c, err := client.New("http://localhost:10001", nil)
    if err != nil {
        fmt.Println("failed to create client:", err)
        return
    }
    output, err := c.RunPlugin("googlesearch", map[string]any{
        "query": "latest Golang news",
    })
    if err != nil {
        fmt.Println("failed to run plugin:", err)
        return
    }
    fmt.Println(output)
}
```