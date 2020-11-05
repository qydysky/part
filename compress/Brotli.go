package part

import (
    br "github.com/andybalholm/brotli"
    "bytes"
    "io"
    "io/ioutil"
)

func InBr(byteS []byte, level int) ([]byte,error) {
    buf := bytes.NewBuffer(nil)
    Write := br.NewWriterLevel(buf, level)
    defer Write.Close()

    // 写入待压缩内容
    if _,err := Write.Write(byteS); err != nil {return buf.Bytes(),err}
    if err := Write.Flush(); err != nil {return buf.Bytes(),err}
    return buf.Bytes(),nil
}

func UnBr(byteS []byte) ([]byte,error) {
    buf := bytes.NewBuffer(byteS)
    Read := br.NewReader(buf)

    rb, err := ioutil.ReadAll(Read)
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        return rb, nil
    }
    return rb, err
}