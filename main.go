package main

import (
	"bufio"
	"flag"
	"github.com/BurntSushi/rure-go"
	"os"
	"sync"
)

var memgrok = flag.String("memgrok", "grok.out", "write memory profile to this file")

type GRegexp struct {
	rure     *rure.Regex
	names    []string
	caps     []*rure.Captures
	isInUsed []bool
	Mutex    sync.Mutex
}

type Config struct {
	gRegexp []*GRegexp
}

var done = make (chan bool,1)

func (gr *GRegexp) unused(i int)  {
	gr.Mutex.Lock()
	gr.isInUsed[i] = false
	gr.Mutex.Unlock()
}

func RureParseTypedCompiled(gr *GRegexp, text string) (map[string]interface{}, error) {
	captures := make(map[string]interface{})
	var noc int
	for i := 0 ; i < 100; i++ {
		gr.Mutex.Lock()
		if !gr.isInUsed[i] {
			gr.isInUsed[i] = true
			noc = i
			gr.Mutex.Unlock()
			defer gr.unused(noc)
			break
		} else {
			gr.Mutex.Unlock()
		}
	}
	if !gr.rure.Captures(gr.caps[noc], text) {
		return nil, nil
	}
	for i, segmentName := range gr.names {
		if segmentName != "" {
			begin, end, ok := gr.caps[noc].Group(i)
			if ok && begin != end {
				captures[segmentName] = text[begin:end]
			}
		}
	}
	return captures, nil
}

func loop(cfg *Config)  {
	var text string
	file, err := os.Open("nginx_access.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text = scanner.Text()
		for _, gr := range cfg.gRegexp {
			values, err := RureParseTypedCompiled(gr, text)
			if err == nil && values != nil {
				break
			}
		}
	}
	done <- true
}

func Compile(pattern string) (*GRegexp, error) {
	// Compile
	re, err := rure.Compile(pattern)
	if err != nil {
		panic(err)
	}
	names := re.CaptureNames()
	gr := &GRegexp{
		rure:re,
		names:names,
	}
	for i:= 0; i<100; i++ {
		gr.caps  = append(gr.caps, gr.rure.NewCaptures())
		gr.isInUsed = append(gr.isInUsed, false)
	}
	return gr, nil
}

func GetPatterns() []string {
	var patterns []string
	file, err := os.Open("web_access_pattern")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		patterns = append(patterns, scanner.Text())
	}
	return patterns
}

func main() {
	// Config
	thread := 4
	patterns := GetPatterns()

	// Compile patterns
	cfg := Config{}
	for _, pattern := range patterns {
		gr, err := Compile(pattern)
		if err != nil {
			panic(err)
		} else {
			cfg.gRegexp = append(cfg.gRegexp, gr)
		}
	}

	// Run
	for i:=0; i<thread; i++ {
		go loop(&cfg)
	}
	<-done
}
