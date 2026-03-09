package zip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

type GzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w GzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type GzipReader struct {
	http.Request
	Reader io.Reader
}

func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	w, err := gzip.NewWriterLevel(&b, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	return b.Bytes(), nil
}

//func Decompress(data []byte) ([]byte, error) {
//	r, err := gzip.NewReader(bytes.NewReader(data))
//	if err != nil {
//		return nil, fmt.Errorf("failed init decompress reader %v", err)
//	}
//	defer r.Close()
//
//	var b bytes.Buffer
//	_, err = b.ReadFrom(r)
//	if err != nil {
//		return nil, fmt.Errorf("failed decompress data: %v", err)
//	}
//
//	return b.Bytes(), nil
//}
