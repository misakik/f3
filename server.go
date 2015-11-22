package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "net/http"
  "github.com/guregu/kami"
  "golang.org/x/net/context"
  "github.com/misakik/f3/search"
)

func main() {
  flag.Parse()

  switch flag.Arg(0) {

    case "index" :
      search.Index(flag.Arg(1))

    case "search":
      search.OpenIndex()

      result, err := search.Search(flag.Arg(1))
      if err != nil {
        fmt.Println(err)
        return
      }
      fmt.Println(result)

    case "server" :
      search.OpenIndex()

      kami.Get("/search/:keyword", searchHandler)
      kami.Serve()
  }
}

func searchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
  keyword := kami.Param(ctx, "keyword")
  result, err := search.Search(keyword)
  if err != nil {
    fmt.Println(err)
    return
  }

  json, _ := json.Marshal(result)
  fmt.Fprintf(w, string(json))
}
