package ghttp

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/panjf2000/gnet/v2"
)

// HandlerFunc handles HTTP Requests
type HandlerFunc = func(req Request, res *Response) error

// Router is the core element of a HTTP Server and navigates
// the requests.
//
// Routes have a specific syntax: @METHOD/path/[option1|option2]/#/*
//
// The @METHOD is optional and can be @GET @POST and so on.
// If @METHOD is not specified every method is applicable for that route.
//
// path is just your average path.
//
// [option1|option2] Can be a group of one ore more options to pick for example:
//
//	@GET/example/[first|second]/test
//
// will match /example/first/test and /example/second/test
//
// The # can be used to descripe a number. The path /example/# will
// match any /example/ path suffixed with a number.
// The method Request.PathInt can be used to get the number.
//
// The * matches anything. The method Request.PathSequence can be used
// to get the data.
type Router struct {
	routes routerRoot
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		[methodCount]branch{
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
			createBranch(),
		},
	}
}

// Register setups the router to handle requests for the given route
func (router *Router) Register(route string, handler HandlerFunc) {
	addBranch(route, &router.routes, handler)
}

func (router *Router) findRoute(r Request, path []byte) HandlerFunc {
	method := r.Method()
	branch := router.routes[method]
	start := 1
	for start < len(path) {
		end := len(path)
		for i := start; i < len(path); i++ {
			if path[i] == '/' {
				end = i
				break
			}
		}
		part := path[start:end]
		maybeBranch, ok := branch.fixed[*unsafeString(&part)]
		if !ok {
			for _, dynamic := range branch.dynamic {
				if dynamic.matcher(part) {
					branch = *dynamic
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}
		} else {
			branch = *maybeBranch
		}
		start = end + 1
	}
	return branch.handler
}

var signalPool = sync.Pool{New: func() any { return new(bool) }}

func (router *Router) call(conn gnet.Conn, p *httpParser, body []byte) bool {
	response := getResponse()

	request := Request{
		conn:     conn,
		parser:   p,
		data:     body,
		detached: signalPool.Get().(*bool),
		response: response,
	}

	handler := router.findRoute(request, p.path)
	hc := conn.Context().(*httpCodec)

	if handler == nil {
		response.status = 404
		response.Write([]byte("Not Found"))
		response.renderResponse(hc.buf)
		returnResponse(response)
		return false
	}

	err := handler(request, response)
	if err != nil {
		response.status = 500
		response.body.Reset()
		response.Write([]byte("Internal Server Error"))
	}

	if *request.detached {
		signalPool.Put(request.detached)
		returnResponse(response)
		return true
	}
	signalPool.Put(request.detached)
	response.renderResponse(hc.buf)
	returnResponse(response)
	return false
}

func unsafeString(b *[]byte) *string {
	return (*string)(unsafe.Pointer(b))
}

type branches struct {
	branches []branch
}

type matcher = func(part []byte) bool
type routerRoot = [methodCount]branch

type branch struct {
	matcher matcher
	fixed   map[string]*branch
	dynamic []*branch
	handler HandlerFunc
}

func methodMatcher(pathParts []string) ([]string, int) {
	if len(pathParts) == 0 {
		return pathParts, MethodUnkown
	}
	if len(pathParts[0]) == 0 || pathParts[0][0] != '@' {
		return pathParts, MethodUnkown
	}
	return pathParts[1:], requestMethod([]byte(strings.ToUpper(pathParts[0])[1:]))
}

func dropEmpty(pathParts []string) []string {
	if len(pathParts) > 0 && pathParts[0] == "" {
		return dropEmpty(pathParts[1:])
	}
	return pathParts
}

func createBranch() branch {
	return branch{fixed: make(map[string]*branch)}
}

func appendStage(parts []string, b *branch) ([]string, *branch) {
	new := createBranch()
	part := parts[0]
	if !(part[0] == '[' && part[len(part)-1] == ']') && part[0] != '#' && part[0] != '*' {
		b.fixed[part] = &new
		return parts[1:], &new
	}
	if part[0] == '[' {
		subBranches := strings.Split(part[1:len(part)-1], "|")
		if len(subBranches) == 0 {
			b.fixed[""] = &new
			return parts[1:], &new
		} else {
			for _, item := range subBranches {
				b.fixed[item] = &new
			}
			return parts[1:], &new
		}
	}
	if part[0] == '#' {
		new.matcher = isNum
		b.dynamic = append(b.dynamic, &new)
		return parts[1:], &new
	}
	if part[0] == '*' {
		new.matcher = alwaysMatch
		b.dynamic = append(b.dynamic, &new)
		return parts[1:], &new
	}
	panic("unreachable")
}

func alwaysMatch([]byte) bool {
	return true
}

func isNum(b []byte) bool {
	for _, b := range b {
		if b > '9' || b < '0' {
			return false
		}
	}
	return true
}

func mergeBranch(b *branch, o *branch) *branch {
	for k, v := range o.fixed {
		if _, ok := b.fixed[k]; ok {
			mergeBranch(b.fixed[k], v)
			continue
		}
		b.fixed[k] = v
	}
	b.handler = o.handler
	b.matcher = o.matcher

	b.dynamic = append(b.dynamic, o.dynamic...)
	return b
}

func addBranch(path string, router *routerRoot, handler HandlerFunc) {
	parts := strings.Split(path, "/")
	parts = dropEmpty(parts)
	parts, methodGuard := methodMatcher(parts)
	parts = dropEmpty(parts)
	b := createBranch()
	branch := &b
	for len(parts) > 0 {
		parts, branch = appendStage(parts, branch)
	}
	branch.handler = handler

	if methodGuard != MethodUnkown {
		router[methodGuard] = *mergeBranch(&router[methodGuard], &b)
	} else {
		for i := 0; i < methodCount; i++ {
			router[i] = *mergeBranch(&router[i], &b)
		}
	}
}
