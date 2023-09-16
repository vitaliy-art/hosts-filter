package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

var (
	inputFile         = "domains.txt"
	outputSuccessFile = "success.txt"
	outputFailsFile   = "fails.txt"
	domainFilterFile  = "domain_filter.txt"
	workersCount      = 100
	accept4XXErrors   = true

	domainFilter = []string{}
)

func main() {
	parseParameters()

	if err := loadDomainFilter(); err != nil {
		panic(err)
	}

	if err := filterDomains(); err != nil {
		panic(err)
	}
}

func parseParameters() {
	flag.StringVar(&inputFile, "input_file", inputFile, "set input file")
	flag.StringVar(&outputSuccessFile, "output_success_file", outputSuccessFile, "set output success file")
	flag.StringVar(&outputFailsFile, "output_fails_file", outputFailsFile, "set output fails file")
	flag.StringVar(&domainFilterFile, "domain_filter_file", domainFilterFile, "set domain filter file")
	flag.IntVar(&workersCount, "workers_count", workersCount, "set workers count")
	flag.BoolVar(&accept4XXErrors, "accept_4xx_errors", accept4XXErrors, "set accept 4XX error; if false, 4XX will be included into fails file")
	flag.Parse()
}

func loadDomainFilter() error {
	f, err := os.Open(domainFilterFile)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			if !strings.HasPrefix(line, ".") {
				line = "." + line
			}

			if !strings.HasSuffix(line, "/") {
				line += "/"
			}

			domainFilter = append(domainFilter, line)
		}
	}

	return scanner.Err()
}

func filterDomains() error {
	fInput, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer fInput.Close()

	fOutputSuccess, err := os.Create(outputSuccessFile)
	if err != nil {
		return err
	}
	defer fOutputSuccess.Close()

	fOutputFails, err := os.Create(outputFailsFile)
	if err != nil {
		return err
	}
	defer fOutputFails.Close()

	var (
		scanner       = bufio.NewScanner(fInput)
		successWriter = bufio.NewWriter(fOutputSuccess)
		failWriter    = bufio.NewWriter(fOutputFails)
	)

	var (
		wg        = &sync.WaitGroup{}
		inputCh   = make(chan string, workersCount)
		failCh    = make(chan string)
		successCh = make(chan string)
	)

	minErrorCode := 400
	if accept4XXErrors {
		minErrorCode = 500
	}

	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go work(inputCh, failCh, successCh, wg, minErrorCode)
	}

	go write(failCh, failWriter)
	go write(successCh, successWriter)

	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("processing %d: %s\n", i, line)
		inputCh <- line
		i++
	}

	close(inputCh)
	wg.Wait()
	close(failCh)
	close(successCh)
	return nil
}

func write(ch <-chan string, w *bufio.Writer) {
	for line := range ch {
		if _, err := w.WriteString(line); err != nil {
			panic(err)
		}
		w.Flush()
	}
}

func work(ch <-chan string, f, s chan<- string, wg *sync.WaitGroup, minErrorCode int) {
	for line := range ch {
		if err := processLine(line, f, s, minErrorCode); err != nil {
			panic(err)
		}
	}

	wg.Done()
}

func processLine(domain string, failCh, successCh chan<- string, minErrorCode int) error {
	line := domain
	if !strings.HasSuffix(line, "/") {
		line += "/"
	}

	available := false
	for _, filter := range domainFilter {
		if strings.HasSuffix(line, filter) {
			available = true
			break
		}
	}

	if !available {
		output := fmt.Sprintf("%s - not available top level domain;\n", domain)
		failCh <- output
		return nil
	}

	url, err := url.Parse(line)
	if err != nil {
		return err
	}

	if url.Scheme == "" {
		url.Scheme = "http"
	}

	response, err := http.DefaultClient.Get(url.String())
	if err != nil || response == nil {
		output := fmt.Errorf("%s - error while get url: %w;\n", domain, err).Error()
		failCh <- output
		return nil
	}
	defer response.Body.Close()

	if response.StatusCode >= minErrorCode {
		output := fmt.Sprintf("%s - error status code %s;\n", domain, response.Status)
		failCh <- output
		return nil
	}

	output := fmt.Sprintf("%s - status code %s;\n", domain, response.Status)
	successCh <- output
	return err
}
