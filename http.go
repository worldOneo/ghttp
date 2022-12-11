package ghttp

import (
	"bufio"
	"bytes"
	"sync"
	"time"

	"github.com/evanphx/wildcat"
	"github.com/panjf2000/gnet/v2"
)

type httpServer struct {
	router *Router
}

var bytePool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

type httpCodec struct {
	parser *wildcat.HTTPParser
	buf    *bytes.Buffer
}

func (hs *httpServer) OnBoot(eng gnet.Engine) (action gnet.Action) {
	return gnet.None
}

func (hs *httpServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	hc := c.Context().(*httpCodec)
	hc.buf.Reset()
	bytePool.Put(hc.buf)
	return gnet.None
}

func (hs *httpServer) OnShutdown(eng gnet.Engine) {
}

func (hs *httpServer) OnTick() (delay time.Duration, action gnet.Action) {
	delay = time.Second
	return
}

func (hs httpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	c.SetContext(&httpCodec{parser: wildcat.NewHTTPParser(), buf: bytePool.Get().(*bytes.Buffer)})
	return nil, gnet.None
}

func (hs *httpServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	hc := c.Context().(*httpCodec)
	body := bytePool.Get().(*bytes.Buffer)
pipeline:
	data, err := c.Next(-1)
	if err != nil && err != bufio.ErrBufferFull {
		return gnet.Close
	}
	headerOffset, err := hc.parser.Parse(data)
	if err == wildcat.ErrMissingData {
		return gnet.None
	}
	if err != nil {
		return gnet.Close
	}

	bodyLen := int(hc.parser.ContentLength())
	if body.Len() < bodyLen {
		body.Write(data[headerOffset:])
	}
	if bodyLen == -1 {
		bodyLen = 0
	}
	data = data[headerOffset+bodyLen:]
	if len(data) > 0 {
		goto pipeline
	}

	hc.buf.Reset()
	detached := hs.router.call(c, hc.parser, body.Bytes())
	body.Reset()
	bytePool.Put(body)
	// handle the request
	if !detached {
		c.Write(hc.buf.Bytes())
	}
	return
}

// StartServer launches and listens as an http server on the given address.
// This will block until an error occurs or the server is terminated.
func StartServer(router *Router, address string) error {
	http := &httpServer{router}
	return gnet.Run(http, address, gnet.WithMulticore(true))
}

var byteSlicePoolSizes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

var slicePools = []sync.Pool{}

func init() {
	for _, size := range byteSlicePoolSizes {
		func(i int) {
			slicePools = append(slicePools, sync.Pool{New: func() any {
				return make([]byte, 1<<(i*2))
			}})
		}(size)
	}
}

// GetSlice provides a byte slice of at least the needed size
// the slices are pooled and therefore it is more efficient than allocations.
// Use ReturnSlice to put the slice back into the pool to prevent a dry out.
func GetSlice(needed int) []byte {
	for i, size := range byteSlicePoolSizes {
		if 1<<(size*2) > needed {
			return slicePools[i].Get().([]byte)
		}
	}
	return make([]byte, needed)
}

// ReturnSlice returns a slice s back to the pool.
func ReturnSlice(s []byte) {
	for i, size := range byteSlicePoolSizes {
		if 1<<(size*2) == cap(s) {
			slicePools[i].Put(s)
		}
	}
}