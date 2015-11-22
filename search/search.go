package search

import (
  "bytes"
  "fmt"
  "io"
  "io/ioutil"
  "mime/multipart"
  "net/http"
  "os"
  "path/filepath"
  "strings"
  "time"

  "github.com/blevesearch/bleve"

  "github.com/misakik/f3/metadata"
)

const IndexDBDir  = ".tmp/index.data"
var LimeIndex bleve.Index
const MinFileSize = 10000000
const TikaURL     = "http://localhost:9998/tika"

type IndexData struct {
  Name string
  Size int64
  IsDir bool
  ModTime time.Time
  Text string
}

func Index(root string) {
  count := 0
  // Delete index directory if it already exists
  if _, err := os.Stat(IndexDBDir); err == nil {
    os.RemoveAll(IndexDBDir)
  }

  mapping := bleve.NewIndexMapping()
  index, err := bleve.New(IndexDBDir, mapping)
  if err != nil {
      fmt.Println(err)
      return
  }

  err = filepath.Walk(root,
    func(path string, f os.FileInfo, err error) error {

      // Skip hidden files
      if strings.HasPrefix(f.Name(), ".") {
          return nil
      }

      count += 1

      size := f.Size()
      text := ""
      isDir:= f.IsDir()
      modTime := f.ModTime()

      if !f.IsDir() && size < MinFileSize {

        err := metadata.WriteMetaData(path, f)
        if err != nil {
          fmt.Println("error writing meta data")
          return err
        }

        client := &http.Client{}

        bodyBuf := &bytes.Buffer{}
        bodyWriter := multipart.NewWriter(bodyBuf)

        fileWriter, err := bodyWriter.CreateFormFile("uploadfile", path)
        if err != nil {
            fmt.Println("error writing to buffer")
            return err
        }

        fh, err := os.Open(path)
        if err != nil {
            fmt.Println("error opening file")
            return err
        }
        defer fh.Close()

        _, err = io.Copy(fileWriter, fh)
        if err != nil {
            return err
        }

        //contentType := bodyWriter.FormDataContentType()
        bodyWriter.Close()

        request, err := http.NewRequest("PUT", TikaURL, bodyBuf)
        if err != nil {
          fmt.Println(err)
          return nil
        }

        request.Header.Set("Accept", "text/plain")

        response, err := client.Do(request)
        if err != nil {
          fmt.Println(err)
          return nil
        }

        defer response.Body.Close()
        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
          fmt.Println(err)
          return nil
        }
        text = string(body)



      }

      //fmt.Printf("%d : Name: %s | Size: %d | IsDir: %t | ModTime: %s | Text: %t \n", count, path, size, isDir, modTime, (len(text) >0))
      data := IndexData{Name: path, Size: size, IsDir: isDir, ModTime: modTime, Text: text}
      index.Index(path, data)
      return nil
    })
  if err != nil {
      fmt.Println(err)
      return
  }

  fmt.Printf("Index Done. %d items.\n", count)
}

func OpenIndex() {
  index, err := bleve.Open(IndexDBDir)
  LimeIndex = index
  if err != nil {
    fmt.Println(err)
    return
  }
}

//See https://godoc.org/github.com/blevesearch/bleve#SearchResult
func Search(keyword string) (*bleve.SearchResult, error) {
  fmt.Println(keyword)

  query := bleve.NewMatchQuery(keyword)
  request := bleve.NewSearchRequest(query)
  request.Size = 20
  //request.Fields = []string{"Name", "Text", "Size", "IsDir", "ModTime"}

  result, err := LimeIndex.Search(request)
  if err != nil {
      return nil, err
  }
  return result, nil
}
