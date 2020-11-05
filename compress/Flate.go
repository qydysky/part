package part

import (
    "github.com/klauspost/compress/flate"
    "bytes"
    "io"
    "io/ioutil"
)

func InFlate(byteS []byte, level int) ([]byte,error) {
    buf := bytes.NewBuffer(nil)

    // 创建一个flate.Write
    flateWrite, err := flate.NewWriter(buf, level)
    if err != nil {
        return buf.Bytes(),err
    }
    defer flateWrite.Close()
    // 写入待压缩内容
    flateWrite.Write(byteS)
    flateWrite.Flush()
    return buf.Bytes(),nil
}

func UnFlate(byteS []byte) ([]byte,error) {
    buf := bytes.NewBuffer(byteS)
    // 创建一个flate.Write
    flateRead := flate.NewReader(buf)
    defer flateRead.Close()
    // 写入待压缩内容
    rb, err := ioutil.ReadAll(flateRead)
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        return rb, nil
    }
    return rb, err
}