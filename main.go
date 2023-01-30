package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/fatih/color"
)

var Client *http.Client = &http.Client{}

func main() {
	var laddr = flag.String("l", "127.0.33.1:8080", "listen address")
	var saddr = flag.String("d", "127.0.0.1:5500", "server address")
	flag.Parse()

	http.HandleFunc("/", getProxyFunc(*saddr))
	err := http.ListenAndServe(*laddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getProxyFunc(dst string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get request info from client
		newUrl := r.URL
		newUrl.Host = dst
		newUrl.Scheme = "http"

		log.Println(newUrl.String(), r.Method)

		reqText, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		// log
		color.Yellow(fmt.Sprintln(r.Header))
		color.Green(fmt.Sprintln(string(reqText)))

		req, err := http.NewRequest(r.Method, newUrl.String(), bytes.NewBuffer(reqText))
		if err != nil {
			log.Println(err)
			return
		}

		// make request to server
		resp, err := Client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}

		// send response to client
		// write headers
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		// plain
		w.Header().Del("Content-Encoding")
		w.WriteHeader(resp.StatusCode)

		// get body
		body := getPlainTextReader(resp.Body, resp.Header.Get("Content-Encoding"))
		respText, err := io.ReadAll(body)
		if err != nil {
			log.Println(err)
			return
		}

		// log
		color.Yellow(fmt.Sprintln(resp.Header))
		color.Green(fmt.Sprintln(string(respText)))

		// return to request
		w.Write(respText)
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
