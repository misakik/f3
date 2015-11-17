package main

import (
    "bytes"
    "crypto/sha1"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "mime/multipart"
    "net/http"
    "strings"
    "time"
    "os"
    "os/exec"
    "path/filepath"
    "github.com/blevesearch/bleve"
    //"github.com/gographics/imagick/imagick"
    "github.com/guregu/kami"
    "golang.org/x/net/context"
    "gopkg.in/redis.v3"
)

type IndexData struct {
  Name string
  Size int64
  IsDir bool
  ModTime time.Time
  Text string
}

type MetaData struct {
  IsDir bool
  Size int64
  ModTime time.Time
  Hash string
}

const IndexDir = ".tmp/index.data"
const TikaURL = "http://localhost:9998/tika"
const MinFileSize = 10000000
const QLmanage = "/usr/bin/qlmanage"

var LimeIndex bleve.Index
//var RedisClient redis.Client

func init() {


  //RedisClient.FlushAll()
  //fmt.Println("Flushed DB No.9")
}

// Create thumbnail from path.
func MakeThumb(path string) (image []byte) {
  tmpdir, err := ioutil.TempDir("", "Limelight-")
  if err != nil {
    fmt.Println(err)
    return
  }
  defer os.RemoveAll(tmpdir)
  exec.Command(QLmanage, path, "-t", "-o", tmpdir).Run()

  // Write here saving thumb
  return nil
}


func WriteMetaData(path string, f os.FileInfo) (error) {
  size := f.Size()
  isDir:= f.IsDir()
  modTime := f.ModTime()

  var result []byte
  file, err := os.Open(path)
  if err != nil {
    return err
  }
  defer file.Close()

  hash := sha1.New()
  if _, err := io.Copy(hash, file); err != nil {
    return err
  }

  s := hash.Sum(result)

  data := MetaData{IsDir: isDir, Size: size, ModTime: modTime, Hash: fmt.Sprintf("%x", s)}
  json, _ := json.Marshal(data)

  RedisClient := redis.NewClient(&redis.Options{
      Addr:     "localhost:16379",
      Password: "", // no password set
      DB:       9,  // use DB No.9
  })
  defer RedisClient.Close()

  ero := RedisClient.Set(path, json, 0).Err()
  if ero != nil {
      panic(ero)
  }

  fmt.Printf("%s : %x\n", path, s)

  return nil
}

func main() {
  flag.Parse()

  switch flag.Arg(0) {
  case "index" :
    count := 0
    // Delete index directory if it already exists
    if _, err := os.Stat(IndexDir); err == nil {
      os.RemoveAll(IndexDir)
    }

    mapping := bleve.NewIndexMapping()
    index, err := bleve.New(IndexDir, mapping)
    if err != nil {
        fmt.Println(err)
        return
    }

    root := flag.Arg(1)
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

          err := WriteMetaData(path, f)
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

  case "search":
    result, err := Search(flag.Arg(1))
    if err != nil {
      fmt.Println(err)
      return
    }
    fmt.Println(result)

  case "server" :
    index, err := bleve.Open(IndexDir)
    LimeIndex = index
    if err != nil {
      fmt.Println(err)
      return
    }

    kami.Get("/search/:keyword", searchHandler)
    kami.Serve()
  }
}

func searchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
  keyword := kami.Param(ctx, "keyword")
  result, err := Search(keyword)
  if err != nil {
    fmt.Println(err)
    return
  }

  json, _ := json.Marshal(result)
  fmt.Fprintf(w, string(json))
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
