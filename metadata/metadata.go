package metadata

import(
  "crypto/sha1"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "time"

  "gopkg.in/redis.v3"
  //"github.com/gographics/imagick/imagick"
)

const QLmanage = "/usr/bin/qlmanage"

type MetaData struct {
  IsDir bool
  Size int64
  ModTime time.Time
  Hash string
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
