package ghttp

import (
	"bytes"
	"errors"
)

type pair = [2][]byte

type httpParser struct {
	version       int
	method        int
	contentLength int64
	path          []byte
	query         []pair
	header        []pair
}

// HTTP Methods
const (
	MethodGet = iota
	MethodHead
	MethodPost
	MethodPut
	MethodPatch
	MethodDelete
	MethodConnect
	MethodOptions
	MethodTrace
	MethodUnkown
	methodCount
)

// HTTP Protocoll
const (
	HTTP0_9 = iota
	HTTP1_0
	HTTP1_1
)

const (
	parserDefaultQuery  = 3
	parserDefaultHeader = 10
)

func NewHTTPParser() *httpParser {
	return &httpParser{
		query:  make([]pair, parserDefaultQuery),
		header: make([]pair, parserDefaultHeader),
	}
}

func requestMethod(data []byte) int {
	l := len(data)
	if l == 3 {
		if string(data) == "GET" {
			return MethodGet
		}
		if string(data) == "PUT" {
			return MethodPut
		}
	} else if l == 4 {
		if string(data) == "HEAD" {
			return MethodHead
		}
		if string(data) == "POST" {
			return MethodPost
		}
	} else if l == 5 {
		if string(data) == "PATCH" {
			return MethodPatch
		}
		if string(data) == "TRACE" {
			return MethodTrace
		}
	} else if l == 6 {
		if string(data) == "DELETE" {
			return MethodDelete
		}
	} else if l == 7 {
		if string(data) == "CONNECT" {
			return MethodConnect
		}
		if string(data) == "OPTIONS" {
			return MethodOptions
		}
	}
	return MethodUnkown
}

func isHorSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

var ErrIncompleteData = errors.New("incomplete http request")
var ErrBadData = errors.New("invalid http request")
var ErrUnsupportedMethod = errors.New("http method not supported")
var ErrUnsupportedProtocol = errors.New("protocol not supported")

var shortestRequestPossible = []byte("GET / HTTP/X.X\r\n\r\n")
var minRequestSize = len(shortestRequestPossible)

func (hp *httpParser) Parse(content []byte) (int, error) {
	hp.contentLength = -1
	hp.query = hp.query[:0]
	hp.header = hp.header[:0]
	if len(content) < minRequestSize {
		return 0, ErrIncompleteData
	}
	length := len(content)
	reader := 0
	methodStart := reader
	for reader < length && !isHorSpace(content[reader]) {
		reader++
	}
	if reader == length {
		return 0, ErrIncompleteData
	}
	method := requestMethod(content[methodStart:reader])
	if method == MethodUnkown {
		return 0, ErrUnsupportedMethod
	}
	hp.method = method

	for reader < length && isHorSpace(content[reader]) {
		reader++
	}
	if reader == length {
		return 0, ErrIncompleteData
	}
	queryPathStart := reader
	for reader < length && !isHorSpace(content[reader]) && content[reader] != '?' {
		reader++
	}
	if reader == length {
		return 0, ErrIncompleteData
	}
	queryPathEnd := reader
	hp.path = content[queryPathStart:queryPathEnd]
	if content[reader] == '?' && reader < len(content)-1 {
		for !isHorSpace(content[reader]) {
			reader++
			paramNameStart := reader
			for reader < length && !isHorSpace(content[reader]) && content[reader] != '=' {
				reader++
			}

			if reader == length {
				return 0, ErrIncompleteData
			}
			paramNameEnd := reader
			reader++
			paramValStart := reader
			for reader < length && !isHorSpace(content[reader]) && content[reader] != '&' {
				reader++
			}
			if reader == length {
				return 0, ErrIncompleteData
			}
			paramValEnd := reader
			name := content[paramNameStart:paramNameEnd]
			val := content[paramValStart:paramValEnd]
			hp.query = append(hp.query, pair{name, val})
		}
	}
	for reader < length && !isHorSpace(content[reader]) {
		reader++
	}
	if length < reader+6 {
		return 0, ErrIncompleteData
	}
	if string(content[reader:reader+6]) != " HTTP/" {
		return 0, ErrBadData
	}
	reader += 6
	vStart := reader
	for reader < length && content[reader] != '\r' {
		reader++
	}
	if length < reader+1 {
		return 0, ErrIncompleteData
	}
	if content[reader] != '\r' || content[reader+1] != '\n' {
		return 0, ErrBadData
	}
	vEnd := reader
	version := content[vStart:vEnd]
	if string(version) == "0.9" {
		hp.version = HTTP0_9
	} else if string(version) == "1.0" {
		hp.version = HTTP1_0
	} else if string(version) == "1.1" {
		hp.version = HTTP1_1
	} else {
		return 0, ErrUnsupportedMethod
	}
	reader += 2
	breakEaten := false
	for reader < length && content[reader] != '\r' {
		paramNameStart := reader
		for reader < length && content[reader] != ':' {
			reader++
		}
		paramNameEnd := reader
		reader++
		for reader < length && isHorSpace(content[reader]) {
			reader++
		}
		if reader == length {
			return 0, ErrIncompleteData
		}
		paramValStart := reader
		for reader < length && content[reader] != '\r' {
			reader++
		}
		if reader == length {
			return 0, ErrIncompleteData
		}
		paramValEnd := reader
		reader++
		if content[reader] != '\n' {
			return 0, ErrBadData
		}
		reader++
		breakEaten = true
		name := content[paramNameStart:paramNameEnd]
		val := content[paramValStart:paramValEnd]
		if bytes.EqualFold(contentLength, name) {
			hp.contentLength = BytesToInt(val)
		}
		hp.header = append(hp.header, pair{name, val})
	}
	if reader == length {
		return 0, ErrIncompleteData
	}
	if !breakEaten {
		if length < reader+4 {
			return 0, ErrIncompleteData
		}
		if content[reader] != '\r' || content[reader+1] != '\n' || content[reader+2] != '\r' || content[reader+3] != '\n' {
			return 0, ErrBadData
		}
		return reader + 4, nil

	}
	if length < reader+2 {
		return 0, ErrIncompleteData
	}
	if content[reader] != '\r' || content[reader+1] != '\n' {
		return 0, ErrBadData
	}
	return reader + 2, nil
}

func (hp *httpParser) FindHeader(header []byte) []byte {
	for _, pair := range hp.header {
		if bytes.EqualFold(pair[0], header) {
			return pair[1]
		}
	}
	return nil
}
