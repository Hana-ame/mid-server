package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/andybalholm/brotli"
	"github.com/fatih/color"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var Client *http.Client = &http.Client{}

func main() {

	log.Println("Starting")

	var laddr = flag.String("l", "0.0.0.0:5000", "listen address")
	var saddr = flag.String("d", "127.0.0.1:3000", "server address")
	var reExp = flag.String("r", ".*", "regex to match")
	flag.Parse()

	app := fiber.New()

	wsHandler := websocket.New(func(c *websocket.Conn) {
		// c.Locals is added to the *websocket.Conn
		// log.Println(c.Locals("allowed"))  // true
		// log.Println(c.Params("id"))       // 123
		// log.Println(c.Query("v"))         // 1.0
		// log.Println(c.Cookies("session")) // ""

		// websocket.Conn bindings https://pkg.go.dev/github.com/fasthttp/websocket?tab=doc#pkg-index
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", msg)

			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println("write:", err)
				break
			}
		}
	})

	proxyHandler := getProxyFunc(*saddr, *reExp)

	app.Use("/", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			// log.Println("ws")
			c.Locals("allowed", true)
			return wsHandler(c)
		}
		return c.Next()
	})

	app.All("/*", proxyHandler)

	log.Fatal(app.Listen(*laddr))
	// Access the websocket server: ws://localhost:3000/ws/123?v=1.0
	// https://www.websocket.org/echo.html
}

func getProxyFunc(dst, reg string) func(c *fiber.Ctx) error {

	re, err := regexp.Compile(reg)
	if err != nil {
		log.Fatal(err)
	}
	// isMatch := func(s string) {
	// 	return re.MatchString([]byte(s))
	// }

	return func(c *fiber.Ctx) error {
		// log.Println("http")
		// get request info from client
		// log.Println(c.Request().URI().Host())
		trueHost := c.Request().URI().Host()

		newUrl, err := url.Parse(c.Request().URI().String())
		if err != nil {
			return err
		}
		newUrl.Host = dst
		newUrl.Scheme = "http"

		log.Println(c.Method(), newUrl.String())

		reqText := c.Body()

		// log
		if re.MatchString(newUrl.String()) {
			color.White("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
			color.Yellow(fmt.Sprintln(c.GetReqHeaders()))
			color.Green(fmt.Sprintln(string(reqText)))
		}

		req, err := http.NewRequest(c.Method(), newUrl.String(), bytes.NewBuffer(reqText))
		if err != nil {
			return err
		}

		// forgeted to add Header, how could i do it.
		for k, v := range c.GetReqHeaders() {
			req.Header.Add(k, v)
		}

		// req.Header.Set("X-Host", "n.tsukishima.top") // a temporary solution // not work
		req.Host = "n.tsukishima.top"
		req.Host = string(trueHost)

		// make request to server
		resp, err := Client.Do(req)
		if err != nil {
			return err
		}

		// send response to client
		// write headers
		for k, v := range resp.Header {
			if k != "Content-Encoding" {
				c.Set(k, v[0])
			}
		}
		// plain
		c.Status(resp.StatusCode)

		// get body
		body := getPlainTextReader(resp.Body, resp.Header.Get("Content-Encoding"))
		respText, err := io.ReadAll(body)
		if err != nil {
			return err
		}

		// log
		if re.MatchString(newUrl.String()) {
			color.White(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
			color.Yellow(fmt.Sprintln(resp.Header))
			color.Green(fmt.Sprintln(string(respText)))
		}

		// return to request
		_, err = c.Write(respText)

		return err
	}
}

func getPlainTextReader(body io.ReadCloser, encoding string) io.ReadCloser {
	switch encoding {
	case "gzip":
		reader, err := gzip.NewReader(body)
		if err != nil {
			log.Println("error decoding gzip response", reader)
			log.Println("will return raw body")
			return body
		}
		return reader
	case "br":
		reader := brotli.NewReader(body)
		if reader == nil {
			log.Println("error decoding br response", reader)
			log.Println("will return raw body")
			return body
		}
		return io.NopCloser(reader)
	default:
		return body
	}
}
