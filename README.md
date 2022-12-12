# ghttp - Zero Allocation HTTP Server for Millions of Requests

| | Fasthttp | ghttp |
| --- | --- | --- |
|req/sec| 360k req/sec | 400k req/sec |

These benchmarks were performent on my machine and are nanobencmarks.
This Project only exists to show the potential of a well tuned server.
If youd like to try the benchmarks for yourself:
Fasthttp:
```go
package main

import (
        "github.com/valyala/fasthttp"
)

func main() {
        msg := []byte("Hello, World!")
        requestHandler := func(ctx *fasthttp.RequestCtx) {
                ctx.Write(msg)
        }
        fasthttp.ListenAndServe("localhost:8090", requestHandler)
}
```
ghttp:
```go
package main

import (
	"fmt"
	"log"

	"github.com/worldOneo/ghttp"
)

func main() {
	router := ghttp.NewRouter()
	router.Register("@GET/greet/*", func(req ghttp.Request, res *ghttp.Response) error {
		res.WriteString(fmt.Sprintf("Hello %s", req.PathSequence(1)))
		return nil
	})
	log.Fatal(ghttp.StartServer(router, "localhost:8080"))
}
```
The benchmark was performed with wrk: `$ wrk http://localhost:8080/greet/joshua -c 400 -t 12 --latency`