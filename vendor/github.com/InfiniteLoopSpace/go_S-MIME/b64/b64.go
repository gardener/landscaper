//Package b64 encodes base64 and does formating for S/MIME
package b64

import (
	"bytes"
	"encoding/base64"
	"io"
)

// From https://golang.org/src/encoding/pem/pem.go

//Takes an byteslice and outputs the base64 encoded version with linebreaks. The constant PemLineLength defines the lenght of the lines.
func EncodeBase64(in []byte) ([]byte, error) {

	var buf bytes.Buffer
	var breaker lineBreaker
	breaker.out = &buf

	b64 := base64.NewEncoder(base64.StdEncoding, &breaker)
	if _, err := b64.Write(in); err != nil {
		return nil, err
	}
	b64.Close()
	breaker.Close()

	return buf.Bytes(), nil

}

//Lenght of the lines in the output of the function EncodeBase64.
const PemLineLength = 64

type lineBreaker struct {
	line [PemLineLength]byte
	used int
	out  io.Writer
}

var nl = []byte{'\n'}

func (l *lineBreaker) Write(b []byte) (n int, err error) {
	if l.used+len(b) < PemLineLength {
		copy(l.line[l.used:], b)
		l.used += len(b)
		return len(b), nil
	}

	n, err = l.out.Write(l.line[0:l.used])
	if err != nil {
		return
	}
	excess := PemLineLength - l.used
	l.used = 0

	n, err = l.out.Write(b[0:excess])
	if err != nil {
		return
	}

	n, err = l.out.Write(nl)
	if err != nil {
		return
	}

	return l.Write(b[excess:])
}

func (l *lineBreaker) Close() (err error) {
	if l.used > 0 {
		_, err = l.out.Write(l.line[0:l.used])
		if err != nil {
			return
		}
		_, err = l.out.Write(nl)
	}

	return
}
