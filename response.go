package ghttp

import (
	"bytes"
	"net/http"
	"strconv"
	"sync"
)

// Response descripes an HTTP response.
type Response struct {
	status  int
	body    bytes.Buffer
	headers [][2]string
}

// Write appends the bytes b to the response body.
func (r *Response) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// WriteString appends the string s to the response body.
func (r *Response) WriteString(s string) *Response {
	r.body.WriteString(s)
	return r
}

// Status sets the status code. This can be done multiple times.
func (r *Response) Status(code int) *Response {
	r.status = code
	return r
}

// AddHeader adds the key value pair of {key value} to the header.
func (r *Response) AddHeader(header [2]string) *Response {
	r.headers = append(r.headers, header)
	return r
}

func (r *Response) renderResponse(into *bytes.Buffer) {
	into.WriteString("HTTP/1.1 ")
	into.WriteString(strconv.Itoa(r.status))
	into.WriteByte(' ')
	into.WriteString(http.StatusText(r.status))
	into.WriteString("\r\nServer: ghttp over gnet\r\n")
	for _, header := range r.headers {
		into.WriteString(header[0])
		into.WriteString(": ")
		into.WriteString(header[1])
		into.WriteString("\r\n")
	}
	into.WriteString("Content-Length: ")
	into.WriteString(strconv.Itoa(r.body.Len()))
	into.WriteString("\r\n\r\n")
	into.Write(r.body.Bytes())
}

var responsePool = sync.Pool{
	New: func() any {
		return &Response{
			status:  200,
			body:    bytes.Buffer{},
			headers: [][2]string{},
		}
	},
}

func returnResponse(resp *Response) {
	resp.status = 200
	resp.body.Reset()
	resp.headers = resp.headers[:0]
	responsePool.Put(resp)
}

func getResponse() *Response {
	return responsePool.Get().(*Response)
}
