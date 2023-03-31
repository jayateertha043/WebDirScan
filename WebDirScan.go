package main

import (
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	dirPath, domain, outputDir, httpPortsStr, httpsPortsStr *string
	httpPorts, httpsPorts                                   []int
	verbose                                                 *bool
	threads, timeout                                        *int
	dirs, files                                             []string
	headers                                                 = make(map[string]string)
)

func containsPort(p int, ports []int) bool {
	for _, pu := range ports {
		if pu == p {
			return true
		}
	}
	return false
}

func main() {

	dirPath = flag.String("dir", ".", "Input Directory")
	domain = flag.String("domain", "localhost", "Enter domain to scan")
	httpPortsStr = flag.String("http", "", "Enter HTTP ports (comma-separated)")
	httpsPortsStr = flag.String("https", "", "Enter HTTPS ports (comma-separated)")
	threads = flag.Int("threads", 100, "Number of Threads")
	timeout = flag.Int("timeout", 10, "Timeout for Request in Seconds")
	headersF := flag.String("headers", "", "To use Custom Headers headers.json file")
	outputDir = flag.String("out", ".", "Output Directory")
	verbose = flag.Bool("verbose", false, "Verbose Output")
	printBanner()
	flag.Parse()

	var wg sync.WaitGroup

	if strings.TrimSpace(*httpPortsStr) != "" {
		for _, port := range strings.Split(*httpPortsStr, ",") {
			p, err := strconv.Atoi(port)
			if err != nil {
				panic(err)
			}
			httpPorts = append(httpPorts, p)
		}
	}

	if strings.TrimSpace(*httpsPortsStr) != "" {
		for _, port := range strings.Split(*httpsPortsStr, ",") {
			p, err := strconv.Atoi(port)
			if err != nil {
				panic(err)
			}
			httpsPorts = append(httpsPorts, p)
		}
	}

	if len(httpPorts) == 0 && len(httpsPorts) == 0 {
		fmt.Println("Error: at least one HTTP or HTTPS port must be provided")
		os.Exit(1)
	}

	*headersF = strings.TrimSpace(*headersF)
	if *headersF != "" {
		jsonFile, err := os.Open(*headersF)
		if err != nil {
			fmt.Println("Unable to find " + *headersF)
			return
		}
		defer jsonFile.Close()
		byteValue, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			fmt.Println("Unable to read")
			return
		}
		err = json.Unmarshal(byteValue, &headers)
		if err != nil {
			fmt.Println("Json format invalid in headers.json")
			return
		}
	}
	if *outputDir != "" {

		if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
			os.Mkdir(*outputDir, os.ModePerm)
		}
	} else {
		*outputDir = "."
	}

	filepath.Walk(*dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirs = append(dirs, strings.TrimPrefix(path, *dirPath))
		} else {
			files = append(files, strings.TrimPrefix(path, *dirPath))
		}
		return nil
	})

	fmt.Println("[+] Total Dirs:\t", len(dirs))
	fmt.Println("[+] Total Files:\t", len(files))

	for _, port := range append(httpPorts, httpsPorts...) {
		dirFileName := fmt.Sprintf("%s/dirs_%d.csv", *outputDir, port)
		dirFile, _ := os.Create(dirFileName)
		defer dirFile.Close()
		dirWriter := csv.NewWriter(dirFile)
		defer dirWriter.Flush()
		dirWriter.Write([]string{"URI", "Redirected URI", "Status", "Final Status", "Content Length"})

		fileFileName := fmt.Sprintf("%s/files_%d.csv", *outputDir, port)
		fileFile, _ := os.Create(fileFileName)
		defer fileFile.Close()
		fileWriter := csv.NewWriter(fileFile)
		defer fileWriter.Flush()
		fileWriter.Write([]string{"URI", "Redirected URI", "Status", "Final Status", "Content Length"})

		makeRequest := func(uri string, writer *csv.Writer, wg *sync.WaitGroup, mu *sync.Mutex) {
			defer wg.Done()
			httporhttps := "http"

			if containsPort(port, httpsPorts) {
				httporhttps = "https"
			}
			uri = fmt.Sprintf("%s://%s:%d%s", httporhttps, *domain, port, strings.ReplaceAll(uri, "\\", "/"))
			redirectStatusCodes := make([]int, 0)
			statusCode := ""
			redirects := &LogRedirects{}
			client := http.Client{
				Timeout:   time.Duration(*timeout) * time.Second,
				Transport: redirects,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					redirectStatusCodes = redirects.Codes
					viaLen := len(redirectStatusCodes)
					//fmt.Println(redirects.Codes)
					if viaLen >= 5 {
						return errors.New("too many redirects")
					}

					return nil
				},
			}
			mu.Lock()
			req, err := http.NewRequest("GET", uri, nil)
			if err != nil {
				return
			}
			req.Close = true
			//Set Default Headers with Ramdom UA
			tmpHeaders := BuildDefaultHeaders()
			for key, value := range tmpHeaders {
				req.Header.Set(key, value)
			}

			if headers != nil {
				for key, value := range headers {
					req.Header.Set(key, value)
				}
			}
			mu.Unlock()
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[-] Error making request to %s: %v\n", uri, err)
				//return
			} else {
				if len(redirectStatusCodes) >= 1 {
					statusCode = strconv.Itoa(redirectStatusCodes[0])
				} else {
					statusCode = strconv.Itoa(resp.StatusCode)
				}

				/*if statusCode >= "300" && statusCode <= "400" {
					fmt.Println(uri, statusCode)
				}*/
				finaluri := resp.Request.URL.String()
				finalStatusCode := ""
				finalContentLength := ""

				if url.PathEscape(uri) != url.PathEscape(finaluri) {
					mu.Lock()
					req, err := http.NewRequest("GET", finaluri, nil)
					if err != nil {
						return
					}
					req.Close = true
					//Set Default Headers with Ramdom UA
					tmpHeaders := BuildDefaultHeaders()
					for key, value := range tmpHeaders {
						req.Header.Set(key, value)
					}

					if headers != nil {
						for key, value := range headers {
							req.Header.Set(key, value)
						}
					}
					mu.Unlock()
					finalResp, err := client.Do(req)
					if err != nil {
						fmt.Printf("[-] Error making request to %s: %v\n", finaluri, err)
						//return
					} else {
						finalStatusCode = strconv.Itoa(finalResp.StatusCode)
						finalContentLength = strconv.FormatInt(finalResp.ContentLength, 10)
					}
				}

				u1, err1 := url.Parse(uri)
				u2, err2 := url.Parse(finaluri)
				if err1 == nil || err2 == nil {
					if u1.Path == u2.Path {
						finaluri = ""
						finalStatusCode = ""
					}

					if finalContentLength == "" {
						finalContentLength = strconv.FormatInt(resp.ContentLength, 10)
					}
					mu.Lock()
					defer mu.Unlock()
					if *verbose {
						fmt.Println([]string{uri, finaluri, statusCode, finalStatusCode, finalContentLength})
					}

					writer.Write([]string{uri, finaluri, statusCode, finalStatusCode, finalContentLength})
				}

			}

		}
		const maxPending = 9999 // maximum number of pending requests
		if *threads > maxPending {
			*threads = maxPending
		}

		sem := make(chan struct{}, *threads)
		var mu sync.Mutex

		fmt.Println("[*] Directory Burster Running")
		for _, dir := range dirs {
			sem <- struct{}{}
			wg.Add(1)
			go func(dir string, w *csv.Writer) {
				defer func() {
					<-sem
				}()
				makeRequest(dir, w, &wg, &mu)
			}(dir, dirWriter)
		}

		fmt.Println("[*] Files Burster Running")
		for _, file := range files {
			sem <- struct{}{}
			wg.Add(1)
			go func(file string, w *csv.Writer) {
				defer func() {
					<-sem
				}()
				makeRequest(file, fileWriter, &wg, &mu)
			}(file, fileWriter)
		}
	}
	wg.Wait()
	fmt.Println("[*] Completed !!!")
}

type LogRedirects struct {
	Transport http.RoundTripper
	Codes     []int
}

func (l *LogRedirects) RoundTrip(req *http.Request) (*http.Response, error) {
	if l.Transport == nil {
		l.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := l.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	l.Codes = append(l.Codes, resp.StatusCode)
	return resp, err
}

func printBanner() {
	var Banner string = `

	_______  ______   ______  _________ _______  _______  _______  _______  _       
	|\     /|(  ____ \(  ___ \ (  __  \ \__   __/(  ____ )(  ____ \(  ____ \(  ___  )( (    /|
	| )   ( || (    \/| (   ) )| (  \  )   ) (   | (    )|| (    \/| (    \/| (   ) ||  \  ( |
	| | _ | || (__    | (__/ / | |   ) |   | |   | (____)|| (_____ | |      | (___) ||   \ | |
	| |( )| ||  __)   |  __ (  | |   | |   | |   |     __)(_____  )| |      |  ___  || (\ \) |
	| || || || (      | (  \ \ | |   ) |   | |   | (\ (         ) || |      | (   ) || | \   |
	| () () || (____/\| )___) )| (__/  )___) (___| ) \ \__/\____) || (____/\| )   ( || )  \  |
	(_______)(_______/|/ \___/ (______/ \_______/|/   \__/\_______)(_______/|/     \||/    )_)
																							  
	

   `
	fmt.Println(Banner)
}

// Build Default Headers sent with every request
func BuildDefaultHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      RandomUA(),
		"Accept":          "text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		//	"Accept-Encoding": "gzip,deflate",
		"DNT": "1",
		//"Connection": "close",
	}
}

var userAgents = []string{

	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
}

func RandomUA() string {
	rand.Seed(time.Now().Unix())
	choice := rand.Intn(len(userAgents))
	return userAgents[choice]
}
