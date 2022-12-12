package ghttp

import (
	"bytes"

	"github.com/panjf2000/gnet/v2"
)

// Request represents an HTTP Request
// Any data in the request is shared by default therefore
// it should not be shared with other goroutines.
//
// Use CopyBytes or CopyBytes to keep data longer than your request
// or use HandleBlocking to do blocking
// tasks like DB operations to avoid blocking the I/O loop.
type Request struct {
	conn      gnet.Conn
	parser    *httpParser
	data      []byte
	detached  *bool
	response  *Response
}

// Header returns the value of the header name.
//
// To keep this value longer than the request use CopyString.
func (r Request) Header(header string) string {
	return string(r.parser.FindHeader([]byte(header)))
}

var mt []byte

// Body returns the data of the request body.
//
// To keep this value longer than the request use CopyBytes.
func (r Request) Body() []byte {
	return r.data
}

var contentLength = []byte("Content-Length")

// BodyLength returns the length of the request body.
func (r Request) BodyLength() int64 {
	return BytesToInt(r.parser.FindHeader(contentLength))
}

var host = []byte("Host")

// Host returns the request host retrieved from the header parameter "Host"
func (r Request) Host() string {
	header := r.parser.FindHeader(host)
	if header == nil {
		return ""
	}
	return string(header)
}

// Path returns the request path.
//
// To keep this value longer than the request use CopyString.
func (r Request) Path() string {
	return *unsafeString(&r.parser.path)
}

// Method returns the request method of the request
// defined as the constants MethodGet and similar.
func (r Request) Method() int {
	return r.parser.method
}

// HandleBlocking runs this handler outside of the
// I/O loop.
func (r Request) HandleBlocking(fn HandlerFunc) {
	*r.detached = true
	r.data = CopyBytes(r.data)
	r.response.headers = [][2]string{}
	go func() {
		err := fn(r, r.response)
		if err != nil {
			r.response.status = 500
			r.response.body.Reset()
			r.response.Write([]byte("Internal Server Error"))
		}
		bytes := bytePool.Get().(*bytes.Buffer)
		r.response.renderResponse(bytes)
		r.conn.AsyncWrite(bytes.Bytes(), func(c gnet.Conn, err error) error {
			bytes.Reset()
			bytePool.Put(bytes)
			returnResponse(r.response)
			return nil
		})
	}()
}

// PathSequence returns the nth element of the path.
//
// To keep this value longer than the request use CopyBytes.
func (r Request) PathSequence(n int) []byte {
	path := r.parser.path
	start := 1
	count := 0
	for start < len(path) {
		end := len(path)
		for i := start; i < len(path); i++ {
			if path[i] == '/' {
				end = i
				break
			}
		}
		if count == n {
			return path[start:end]
		}
		start = end + 1
		count++
	}
	return []byte{}
}

// Returns the nth element of the path parsed as an int64.
func (r Request) PathInt(n int) int64 {
	return BytesToInt(r.PathSequence(n))
}
