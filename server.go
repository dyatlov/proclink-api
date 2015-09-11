package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dyatlov/go-oembed/oembed"
	"github.com/dyatlov/go-url2oembed/url2oembed"
	"github.com/jeffail/tunny"
)

type workerData struct {
	Status int
	Data   string
}

type apiWorker struct {
	Parser *url2oembed.Parser
}

type apiHandler struct {
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	u := r.FormValue("url")

	w.Header().Set("Server", "ProcLink")
	w.Header().Set("Content-Type", "application/json")

	_, err := url.Parse(u)

	if err != nil {
		log.Printf("Invalid URL provided: %s", u)
		http.Error(w, "{\"status\": \"error\", \"message\":\"Invalid URL\"}", 500)
		return
	}

	result, err := workerPool.SendWork(u)

	if err != nil {
		log.Printf("An unknown error occured: %s", err.Error())

		http.Error(w, "{\"status\": \"error\", \"message\":\"Internal error\"}", 500)
		return
	}

	if data, ok := result.(*workerData); ok {
		w.WriteHeader(data.Status)
		fmt.Fprintln(w, data.Data)
		return
	}

	log.Print("Unable to decode worker result")

	http.Error(w, "{\"status\": \"error\", \"message\":\"Unable to decode worker result\"}", 500)
}

// Use this call to block further jobs if necessary
func (worker *apiWorker) TunnyReady() bool {
	return true
}

// This is where the work actually happens
func (worker *apiWorker) TunnyJob(data interface{}) interface{} {
	if u, ok := data.(string); ok {
		u = strings.Trim(u, "\r\n")

		log.Printf("Got url: %s", u)

		info := worker.Parser.Parse(u)

		if info == nil {
			log.Printf("No info for url: %s", u)

			return &workerData{Status: 404, Data: "{\"status\": \"error\", \"message\":\"Unable to retrieve information form provided url\"}"}
		}
		if info.Status < 300 {
			log.Printf("Url parsed: %s", u)

			return &workerData{Status: 200, Data: info.String()}
		}

		log.Printf("Something weird: %s", u)

		return &workerData{Status: 411, Data: fmt.Sprintf("{\"status\": \"error\", \"message\":\"Unable to obtain data. Status code: %d\"}", info.Status)}
	}

	return &workerData{Status: 500, Data: "{\"status\": \"error\", \"message\":\"Something weird happened\"}"}
}

var workerPool *tunny.WorkPool

func main() {
	providersFile := flag.String("providers_file", "providers.json", "Path to oembed providers json file")
	workerCount := flag.Int64("worker_count", 1000, "Amount of workers to start")
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8000, "Port to listen on")
	maxHTMLBytesToRead := flag.Int64("html_bytes_to_read", 50000, "How much data to read from URL if it's an html page")
	maxBinaryBytesToRead := flag.Int64("binary_bytes_to_read", 4096, "How much data to read from URL if it's NOT an html page")
	waitTimeout := flag.Int("wait_timeout", 3, "How much time to wait for/fetch response from remote server")

	buf, err := ioutil.ReadFile(*providersFile)

	if err != nil {
		panic(err)
	}

	oe := oembed.NewOembed()
	oe.ParseProviders(bytes.NewReader(buf))

	workers := make([]tunny.TunnyWorker, *workerCount)
	for i := range workers {
		p := url2oembed.NewParser(oe)
		p.MaxHTMLBodySize = *maxHTMLBytesToRead
		p.MaxBinaryBodySize = *maxBinaryBytesToRead
		p.WaitTimeout = time.Duration(*waitTimeout) * time.Second
		workers[i] = &(apiWorker{Parser: p})
	}

	pool, err := tunny.CreateCustomPool(workers).Open()

	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	workerPool = pool

	startServer(*host, *port)
}

func startServer(host string, port int) {
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", host, port),
		Handler:        &apiHandler{},
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
