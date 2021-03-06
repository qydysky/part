package part

import (
    "sync"
    "io"
    "os"
    "context"
    "time"
    "strings"
    "bytes"
    "net"
    "net/http"
    "errors"
    "io/ioutil"
    "net/url"
    compress "github.com/qydysky/part/compress"
    pio "github.com/qydysky/part/io"
    signal "github.com/qydysky/part/signal"
    idpool "github.com/qydysky/part/idpool"
    // "encoding/binary"
)

var (
    ConnectTimeoutErr = errors.New("ConnectTimeoutErr")
    ReadTimeoutErr = errors.New("ReadTimeoutErr")
)

type Rval struct {
    Url string
    PostStr string
    Timeout int
    ReadTimeout int
    ConnectTimeout int
    Proxy string
    Retry int
    SleepTime int
    JustResponseCode bool
    Cookies []*http.Cookie

    SaveToPath string
    SaveToChan chan[]byte
    SaveToPipeWriter *io.PipeWriter

    Header map[string]string
}

type Req struct {
    Respon []byte
    Response  *http.Response
    UsedTime time.Duration

    id *idpool.Id
    idp *idpool.Idpool
    cancel *signal.Signal
    sync.Mutex
}

func New() *Req{
    idp := idpool.New()
    return &Req{
        idp:idp,
        id:idp.Get(),
    }
}

// func main(){
//     var _ReqfVal = ReqfVal{
//         Url:url,
//         Proxy:proxy,
// 		Timeout:10,
// 		Retry:2,
//     }
//     Reqf(_ReqfVal)
// }

func (this *Req) Reqf(val Rval) (error) {
    this.Lock()
	defer this.Unlock()

	var returnErr error

	_val := val;
    
    defer func(){
        this.idp.Put(this.id)
    }()

    this.cancel = signal.Init()

	for SleepTime,Retry:=_val.SleepTime,_val.Retry;Retry>=0;Retry-=1 {
        returnErr=this.Reqf_1(_val)
        select {
        case <- this.cancel.WaitC()://cancel
            return returnErr
        default:
            if returnErr==nil {return nil}
        }
        time.Sleep(time.Duration(SleepTime)*time.Millisecond)
    }

	return returnErr
}

func (this *Req) Reqf_1(val Rval) (err error) {

	var (
        Url string = val.Url
        PostStr string = val.PostStr
        Proxy string = val.Proxy
        Timeout int = val.Timeout
        ReadTimeout int = val.ReadTimeout
        ConnectTimeout int = val.ConnectTimeout
        JustResponseCode bool = val.JustResponseCode
        SaveToChan chan[]byte = val.SaveToChan
        SaveToPath string = val.SaveToPath
        SaveToPipeWriter *io.PipeWriter = val.SaveToPipeWriter

        Header map[string]string = val.Header
    )

    var beginTime time.Time = time.Now()

    var client http.Client

    if Header == nil {Header = make(map[string]string)}

    if Proxy!="" {
        proxy := func(_ *http.Request) (*url.URL, error) {
            return url.Parse(Proxy)
        }
        client.Transport = &http.Transport{Proxy: proxy}
    } else {
        client.Transport = &http.Transport{}
    }
    
    if Url==""{return errors.New("Url is \"\"")}

    Method := "GET"
    var body io.Reader
    if len(PostStr) > 0 {
        Method = "POST";
        body = strings.NewReader(PostStr);
        if _,ok := Header["Content-Type"];!ok {Header["Content-Type"] = "application/x-www-form-urlencoded"}
    }

    cx, cancel := context.WithCancel(context.Background())
    if Timeout > 0 {
        cx, _ = context.WithTimeout(cx,time.Duration(Timeout)*time.Millisecond)
    }
    req,e := http.NewRequest(Method, Url, body)
    if e != nil {panic(e)}
    req = req.WithContext(cx)

    var done = make(chan struct{})
    defer close(done)
    go func(){
        select {
        case <- this.cancel.WaitC():cancel()
        case <- done:
        }
    }()
    
    for _,v := range val.Cookies {
        req.AddCookie(v)
    }

    if _,ok := Header["Accept"];!ok {Header["Accept"] = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`}
    if _,ok := Header["Connection"];!ok {Header["Connection"] = "keep-alive"}
    if _,ok := Header["Accept-Encoding"];!ok {Header["Accept-Encoding"] = "gzip, deflate, br"}
    if SaveToPath != "" {Header["Accept-Encoding"] = "identity"}
    if _,ok := Header["User-Agent"];!ok {Header["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36"}

    for k,v := range Header {
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)

    if err != nil {
        return err
    }
    
    var (
        saveToFile func(io.Reader,string)([]byte,error) = func (Body io.Reader,filepath string) (bodyB []byte,err error) {
            out, err := os.Create(filepath + ".dtmp")
            if err != nil {out.Close();return bodyB,err}
            var buf bytes.Buffer

            w := io.MultiWriter(out, &buf)
            if _, err = io.Copy(w, Body); err != nil {out.Close();return bodyB,err}
            out.Close()
            bodyB = buf.Bytes()

            if err = os.RemoveAll(filepath); err != nil {return bodyB,err}
            if err = os.Rename(filepath+".dtmp", filepath); err != nil {return bodyB,err}
            return bodyB,nil
        }
    )
    this.Response = resp
    if !JustResponseCode {
        defer resp.Body.Close()
        if compress_type := resp.Header[`Content-Encoding`];compress_type!=nil &&
        len(compress_type) != 0 && (compress_type[0] == `br` ||
        compress_type[0] == `gzip` ||
        compress_type[0] == `deflate`) {
            var err error
            this.Respon,err = ioutil.ReadAll(resp.Body)
            if err != nil {return err}

            if compress_type := resp.Header[`Content-Encoding`];
            compress_type!=nil && len(compress_type) != 0 {
                switch compress_type[0]{
                case `br`:
                    if tmp,err := compress.UnBr(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                case `gzip`:
                    if tmp,err := compress.UnGzip(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                case `deflate`:
                    if tmp,err := compress.UnFlate(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                default:;
                }
            }
        } else {
            if SaveToPath != "" {
                if bodyB,err := saveToFile(resp.Body, SaveToPath); err != nil {
                    return err
                } else {
                    if len(bodyB) != 0 {
                        if SaveToChan != nil {
                            SaveToChan <- bodyB
                        } else if SaveToPipeWriter != nil {
                            SaveToPipeWriter.Write(bodyB)
                        } else {
                            this.Respon = append(this.Respon,bodyB...)
                        }
                    } else {
                        return io.EOF
                    }

                    if SaveToChan != nil {
                        close(SaveToChan)
                    }
                    if SaveToPipeWriter != nil {
                        SaveToPipeWriter.Close()
                    }
                }
            } else {
                rc,_ := pio.RW2Chan(resp.Body,nil)
                var After = func(ReadTimeout int) (c <-chan time.Time) {
                    if ReadTimeout > 0 {
                        c = time.NewTimer(time.Millisecond*time.Duration(ReadTimeout)).C
                    }
                    return
                }
                
                select {
                case buf :=<- rc:
                    if len(buf) != 0 {
                        if SaveToChan != nil {
                            SaveToChan <- buf
                        } else if SaveToPipeWriter != nil {
                            SaveToPipeWriter.Write(buf)
                        } else {
                            this.Respon = append(this.Respon,buf...)
                        }
                    } else {
                        err = io.EOF
                        return
                    }
                case <-After(ConnectTimeout):
                    err = ConnectTimeoutErr
                    return
                }

                for loop:=true;loop; {
                    select {
                    case buf :=<- rc:
                        if len(buf) != 0 {
                            if SaveToChan != nil {
                                SaveToChan <- buf
                            } else if SaveToPipeWriter != nil {
                                SaveToPipeWriter.Write(buf)
                            } else {
                                this.Respon = append(this.Respon,buf...)
                            }
                        } else {
                            err = io.EOF
                            loop = false
                            break
                        }
                    case <-After(ReadTimeout):
                        err = ReadTimeoutErr
                        loop = false
                        break
                    }
                    if !this.cancel.Islive() {
                        err = context.Canceled
                        loop = false
                        break
                    }
                }
                if SaveToChan != nil {
                    close(SaveToChan)
                }
                if SaveToPipeWriter != nil {
                    SaveToPipeWriter.Close()
                }
            }
        }
    } else {resp.Body.Close()}
    
    this.UsedTime=time.Since(beginTime)
    
    return nil
}

func (t *Req) Cancel(){t.Close()}

func (t *Req) Close(){
    if !t.cancel.Islive() {return}
    t.cancel.Done()
}

func (t *Req) Id() uintptr {
    if t.id == nil {return 0}
    return t.id.Id
}

func IsTimeout(e error) bool {
    if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, ConnectTimeoutErr) || errors.Is(e, ReadTimeoutErr) {return true}
    if net_err,ok := e.(net.Error);ok && net_err.Timeout() {return true}
    if os.IsTimeout(e) {return true}
    return false
}

func IsCancel(e error) bool {
    return errors.Is(e, context.Canceled)
}