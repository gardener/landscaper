package utils

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Gunzip(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)

	var r io.Reader
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}

	var res bytes.Buffer
	_, err = res.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return res.Bytes(), nil
}

func Gzip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}

	if err = gz.Flush(); err != nil {
		return nil, err
	}

	if err = gz.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
