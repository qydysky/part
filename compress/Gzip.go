package part

import (
    gzip "compress/gzip"
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
    Read,err := gzip.NewReader(bytes.NewBuffer(byteS))
    if err != nil {
        return byteS,err
    }
    rb, err := ioutil.ReadAll(Read)
    Read.Close()
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        return append([]byte{},rb...), nil
    }
    return append([]byte{},rb...), err
}