package part

import (
	"bytes"
	"compress/flate"
	"io"
)

func InFlate(byteS []byte, level int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// 创建一个flate.Write
	flateWrite, err := flate.NewWriter(buf, level)
	if err != nil {
		return buf.Bytes(), err
	}
	defer flateWrite.Close()
	// 写入待压缩内容
	flateWrite.Write(byteS)
	flateWrite.Flush()
	return buf.Bytes(), nil
}

func UnFlate(byteS []byte) ([]byte, error) {
	flateRead := flate.NewReader(bytes.NewBuffer(byteS))
	rb, err := io.ReadAll(flateRead)
	flateRead.Close()
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return append([]byte{}, rb...), nil
	}
	return append([]byte{}, rb...), err
}
