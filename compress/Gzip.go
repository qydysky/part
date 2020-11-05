package part

import (
    gzip "github.com/klauspost/pgzip"
    "bytes"
    "io"
    "io/ioutil"
)

func InGzip(byteS []byte, level int) ([]byte,error) {
    buf := bytes.NewBuffer(nil)
    Write,err := gzip.NewWriterLevel(buf, level)
    if err != nil {
        return buf.Bytes(),err
    }
    defer Write.Close()
    // 写入待压缩内容
    if _,err := Write.Write(byteS); err != nil {return buf.Bytes(),err}
    if err := Write.Flush(); err != nil {return buf.Bytes(),err}
    return buf.Bytes(),nil
}

func UnGzip(byteS []byte) ([]byte,error) {
    buf := bytes.NewBuffer(byteS)
    Read,err := gzip.NewReader(buf)
    if err != nil {
        return buf.Bytes(),err
    }
    defer Read.Close()
    rb, err := ioutil.ReadAll(Read)
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        return rb, nil
    }
    return rb, err
}