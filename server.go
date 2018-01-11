package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dyatlov/go-oembed/oembed"
	"github.com/dyatlov/go-url2oembed/url2oembed"
)

type workerData struct {
	Status int
	Data   string
}

type job struct {
	Url    string
	Result chan workerData
}

type apiHandler struct {
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	u := r.FormValue("url")

	w.Header().Set("Server", "ProcLink")
	w.Header().Set("Content-Type", "application/json")

	// to be able to retrieve data from javascript directly
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	_, err := url.Parse(u)

	if err != nil {
		log.Printf("Invalid URL provided: %s", u)
		http.Error(w, "{\"status\": \"error\", \"message\":\"Invalid URL\"}", 500)
		return
	}

	log.Printf("Sending job: %s", u)

	c := make(chan workerData)
	jobPool <- job{Url: u, Result: c}
	data := <-c

	w.WriteHeader(data.Status)
	fmt.Fprintln(w, data.Data)
}

// This is where the work actually happens
func worker(parser *url2oembed.Parser, jobs <-chan job) {
	for {
		params := <-jobs
		u := strings.Trim(params.Url, "\r\n")

		log.Printf("Got url: %s", u)

		info := parser.Parse(u)

		if info == nil {
			log.Printf("No info for url: %s", u)
			params.Result <- workerData{Status: 404, Data: "{\"status\": \"error\", \"message\":\"Unable to retrieve information from provided url\"}"}
		} else if info.Status < 300 {
			log.Printf("Url parsed: %s", u)
			params.Result <- workerData{Status: 200, Data: info.String()}
		} else {
			log.Printf("Something weird: %s", u)
			params.Result <- workerData{Status: 411, Data: fmt.Sprintf("{\"status\": \"error\", \"message\":\"Unable to obtain data. Status code: %d\"}", info.Status)}
		}
	}
}

var jobPool chan job

// stringsToNetworks converts arrays of string representation of IP ranges into []*net.IPnet slice
func stringsToNetworks(ss []string) ([]*net.IPNet, error) {
	var result []*net.IPNet
	for _, s := range ss {
		_, network, err := net.ParseCIDR(s)
		if err != nil {
			return nil, err
		}
		result = append(result, network)
	}

	return result, nil
}

func main() {
	providersFile := flag.String("providers_file", "providers.json", "Path to oembed providers json file")
	workerCount := flag.Int("worker_count", 1000, "Amount of workers to start")
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8000, "Port to listen on")
	maxHTMLBytesToRead := flag.Int64("html_bytes_to_read", 50000, "How much data to read from URL if it's an html page")
	maxBinaryBytesToRead := flag.Int64("binary_bytes_to_read", 4096, "How much data to read from URL if it's NOT an html page")
	waitTimeout := flag.Int("wait_timeout", 7, "How much time to wait for/fetch response from remote server")
	whiteListRanges := flag.String("whitelist_ranges", "", "What IP ranges to allow. Example: 178.25.32.1/8")
	blackListRanges := flag.String("blacklist_ranges", "", "What IP ranges to disallow. Example: 178.25.32.1/8")

	flag.Parse()

	buf, err := ioutil.ReadFile(*providersFile)

	if err != nil {
		panic(err)
	}

	var whiteListNetworks []*net.IPNet
	if len(*whiteListRanges) > 0 {
		if whiteListNetworks, err = stringsToNetworks(strings.Split(*whiteListRanges, " ")); err != nil {
			panic(err)
		}
	}

	var blackListNetworks []*net.IPNet
	if len(*blackListRanges) > 0 {
		if blackListNetworks, err = stringsToNetworks(strings.Split(*blackListRanges, " ")); err != nil {
			panic(err)
		}
	}

	oe := oembed.NewOembed()
	oe.ParseProviders(bytes.NewReader(buf))

	log.Println("Starting workers:", *workerCount)

	jobPool = make(chan job)
	for i := 0; i < *workerCount; i++ {
		p := url2oembed.NewParser(oe)
		p.MaxHTMLBodySize = *maxHTMLBytesToRead
		p.MaxBinaryBodySize = *maxBinaryBytesToRead
		p.WaitTimeout = time.Duration(*waitTimeout) * time.Second
		p.BlacklistedIPNetworks = blackListNetworks
		p.WhitelistedIPNetworks = whiteListNetworks
		go worker(p, jobPool)
	}

	log.Println("All workers started. Starting server on port", *port)

	startServer(*host, *port, *waitTimeout)
}

func startServer(host string, port int, waitTimeout int) {
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", host, port),
		Handler:        &apiHandler{},
		ReadTimeout:    time.Duration(waitTimeout) * time.Second,
		WriteTimeout:   time.Duration(waitTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
