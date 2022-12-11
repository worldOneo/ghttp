package main

import (
	"fmt"
	"log"
	"time"

	"github.com/worldOneo/ghttp"
)

func dateMiddleware(handler ghttp.HandlerFunc) ghttp.HandlerFunc {
	return func(req ghttp.Request, res *ghttp.Response) error {
		res.AddHeader([2]string{"Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")})
		return handler(req, res)
	}
}

func main() {
	router := ghttp.NewRouter()
	router.Register("@GET/1sec", func(req ghttp.Request, res *ghttp.Response) error {
		req.HandleBlocking(dateMiddleware(func(r ghttp.Request, res *ghttp.Response) error {
			time.Sleep(time.Second)
			return nil
		}))
		return nil
	})
	router.Register("@GET/#", func(req ghttp.Request, res *ghttp.Response) error {
		res.
			Status(200).
			AddHeader([2]string{"Data", time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")}).
			Write([]byte(fmt.Sprintf("Hello %s", string(req.PathSequence(0)))))
		return nil
	})
	router.Register("@GET/greet/*", func(req ghttp.Request, res *ghttp.Response) error {
		res.WriteString(fmt.Sprintf("Hello %s", req.PathSequence(1)))
		return nil
	})
	router.Register("@POST/", func(req ghttp.Request, res *ghttp.Response) error {
		res.
			Status(200).
			AddHeader([2]string{"Data", time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")}).
			Write([]byte(fmt.Sprintf("Hello %s", string(req.Body()))))
		return nil
	})
	log.Fatal(ghttp.StartServer(router, "localhost:8080"))
}
