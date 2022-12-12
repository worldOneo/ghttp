package ghttp

// Stolen in good faith from Evan Phoenix licensed under BSD 2-Clause

import (
	"bufio"
	"bytes"
	"net/http"
	"testing"
)

var simple = []byte("GET / HTTP/1.0\r\n\r\n")

func noError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}
}

func assert(t *testing.T, ok bool) {
	if !ok {
		t.Fatalf("Expected ok")
	}
}

func TestParseSimple(t *testing.T) {
	hp := NewHTTPParser()

	n, err := hp.Parse(simple)
	noError(t, err)

	assert(t, n == len(simple))

	assert(t, hp.version == HTTP1_0)

	assert(t, bytes.Equal([]byte("/"), hp.path))
	assert(t, hp.method == MethodGet)
}

func BenchmarkParseSimple(b *testing.B) {
	hp := NewHTTPParser()

	for i := 0; i < b.N; i++ {
		hp.Parse(simple)
	}
}

func BenchmarkNetHTTP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bufio.NewReader(bytes.NewReader(simple))
		http.ReadRequest(buf)
	}
}

var simpleHeaders = []byte("GET / HTTP/1.0\r\nHost: cookie.com\r\n\r\n")

func TestParseSimpleHeaders(t *testing.T) {
	hp := NewHTTPParser()

	_, err := hp.Parse(simpleHeaders)
	noError(t, err)

	assert(t, bytes.Equal([]byte("cookie.com"), hp.FindHeader([]byte("Host"))))
}

func BenchmarkParseSimpleHeaders(b *testing.B) {
	hp := NewHTTPParser()

	for i := 0; i < b.N; i++ {
		hp.Parse(simpleHeaders)
	}
}

var simple3Headers = []byte("GET / HTTP/1.0\r\nHost: cookie.com\r\nDate: foobar\r\nAccept: these/that\r\n\r\n")

func TestParseSimple3Headers(t *testing.T) {
	hp := NewHTTPParser()

	_, err := hp.Parse(simple3Headers)
	noError(t, err)

	assert(t, bytes.Equal([]byte("cookie.com"), hp.FindHeader([]byte("Host"))))
	assert(t, bytes.Equal([]byte("foobar"), hp.FindHeader([]byte("Date"))))
	assert(t, bytes.Equal([]byte("these/that"), hp.FindHeader([]byte("Accept"))))
}

func BenchmarkParseSimple3Headers(b *testing.B) {
	hp := NewHTTPParser()

	for i := 0; i < b.N; i++ {
		hp.Parse(simple3Headers)
	}
}

func BenchmarkNetHTTP3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bufio.NewReader(bytes.NewReader(simple3Headers))
		http.ReadRequest(buf)
	}
}

var short = []byte("GET / HT")

func TestParseMissingData(t *testing.T) {
	hp := NewHTTPParser()

	_, err := hp.Parse(short)

	assert(t, err == ErrIncompleteData)
}

var specialHeaders = []byte("GET / HTTP/1.0\r\nHost: cookie.com\r\nContent-Length: 50\r\n\r\n")

func TestParseSpecialHeaders(t *testing.T) {
	hp := NewHTTPParser()

	_, err := hp.Parse(specialHeaders)
	noError(t, err)

	assert(t, bytes.Equal([]byte("cookie.com"), hp.FindHeader([]byte("Host"))))
	assert(t, 50 == BytesToInt(hp.FindHeader([]byte("Content-Length"))))
}

func TestFindHeaderIgnoresCase(t *testing.T) {
	hp := NewHTTPParser()

	_, err := hp.Parse(specialHeaders)
	noError(t, err)

	assert(t, bytes.Equal([]byte("50"), hp.FindHeader([]byte("content-length"))))
}
