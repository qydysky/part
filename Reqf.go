package part

import (
    "sync"
    "io"
    "os"
    "context"
    "time"
    "strings"
    "net/http"
    "errors"
    "io/ioutil"
    "net/url"
    // "encoding/binary"
)

type Rval struct {
    Url string
    PostStr string
    Timeout int
    Referer string
    Cookie string
    Proxy string
    ContentType string
    Accept string
    AcceptLanguage string
    Connection string
    Retry int
    SleepTime int
    JustResponseCode bool
    SaveToPath string
}

type req struct {
    ResponseCode int
    Respon []byte
    UsedTime time.Duration
    Cancel chan interface{}
    sync.Mutex
}

func Req() *req{
    return &req{}
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

func (this *req) Reqf(val Rval) (error) {
    this.Lock()
	defer this.Unlock()

	var returnErr error

	_val := val;

    if _val.Timeout==0{_val.Timeout=3}

	for ;_val.Retry>=0;_val.Retry-- {
		returnErr=this.Reqf_1(_val)
        if returnErr==nil {break}
        time.Sleep(time.Duration(_val.SleepTime)*time.Millisecond)
    }

	return returnErr
}

func (this *req) Reqf_1(val Rval) (error) {

	var (
        Url string = val.Url
        PostStr string = val.PostStr
        Referer string = val.Referer
        Cookie string = val.Cookie
        Proxy string = val.Proxy
        Accept string = val.Accept
        ContentType string = val.ContentType
        Connection string = val.Connection
        AcceptLanguage string = val.AcceptLanguage
        Timeout int = val.Timeout
        JustResponseCode bool =val.JustResponseCode
        SaveToPath string =val.SaveToPath
    )

    var beginTime time.Time = time.Now()


    var client http.Client

    if Timeout != -1 {
        client.Timeout = time.Duration(Timeout)*time.Second
    }

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
        if ContentType == "" {ContentType = "application/x-www-form-urlencoded"}
    }

    cx, cancel := context.WithCancel(context.Background())
    req,_ := http.NewRequest(Method, Url, body)
    req = req.WithContext(cx)

    go func(){
        this.Cancel = make(chan interface{})
        <- this.Cancel
        cancel()
    }()

    if Accept==""{Accept = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`}
    if Connection==""{Connection = "keep-alive"}

    if Cookie!=""{req.Header.Add("Cookie",Cookie)}
    if Referer!=""{req.Header.Add("Referer",Referer)}
    if Connection!=""{req.Header.Set("Connection",Connection)}
    if AcceptLanguage!=""{req.Header.Set("Accept-Language",AcceptLanguage)}
    if Accept!=""{req.Header.Add("Accept",Accept)}
    if ContentType!=""{req.Header.Set("Content-Type", ContentType)}  //添加请求头

    req.Header.Add("User-Agent","Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36")
    
    resp, err := client.Do(req)

    if err != nil {
        return err
    }
    
    var saveToFile func(io.Reader,string)error = func (Body io.Reader,filepath string) error {
        out, err := os.Create(filepath + ".dtmp")
        if err != nil {out.Close();return err}

        // resp, err := http.Get(url)
        // if err != nil {out.Close();return err}
        // defer resp.Body.Close()

        if _, err = io.Copy(out, Body); err != nil {out.Close();return err}
        out.Close()

        if err = os.Rename(filepath+".dtmp", filepath); err != nil {return err}
        return nil
    }
    this.ResponseCode = resp.StatusCode
    if !JustResponseCode {
        defer resp.Body.Close()
        if SaveToPath != "" {
            if err := saveToFile(resp.Body, SaveToPath); err != nil {
                return err
            }
        }else{
            this.Respon,_ = ioutil.ReadAll(resp.Body)
        }
    }
    
    this.UsedTime=time.Since(beginTime)
    
    return nil
}