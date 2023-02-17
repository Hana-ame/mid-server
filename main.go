package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/andybalholm/brotli"
	"github.com/fatih/color"
)

var Client *http.Client = &http.Client{}

func main() {
	log.Println("Starting")

	var laddr = flag.String("l", "0.0.0.0:5000", "listen address")
	var saddr = flag.String("d", "127.0.0.1:3000", "server address")
	var reExp = flag.String("r", ".*", "regex to match")
	flag.Parse()

	http.HandleFunc("/", getProxyFunc(*saddr, *reExp))
	err := http.ListenAndServe(*laddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getProxyFunc(dst, reg string) http.HandlerFunc {

	re, err := regexp.Compile(reg)
	if err != nil {
		log.Fatal(err)
	}
	// isMatch := func(s string) {
	// 	return re.MatchString([]byte(s))
	// }

	return func(w http.ResponseWriter, r *http.Request) {
		// get request info from client
		newUrl := r.URL
		newUrl.Host = dst
		newUrl.Scheme = "http"

		log.Println(r.Method, newUrl.String())

		reqText, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		// log
		if re.MatchString(newUrl.String()) {
			color.White("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
			color.Yellow(fmt.Sprintln(r.Header))
			color.Green(fmt.Sprintln(string(reqText)))
		}

		req, err := http.NewRequest(r.Method, newUrl.String(), bytes.NewBuffer(reqText))
		if err != nil {
			log.Println(err)
			return
		}

		// forgeted to add Header, how could i do it.
		for k, v := range r.Header {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}

		// req.Header.Set("X-Host", "n.tsukishima.top") // a temporary solution // not work
		req.Host = "n.tsukishima.top"

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
		if re.MatchString(newUrl.String()) {
			color.White(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
			color.Yellow(fmt.Sprintln(resp.Header))
			color.Green(fmt.Sprintln(string(respText)))
		}

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
