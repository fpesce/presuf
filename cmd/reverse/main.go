/*
Copyright 2020 Francois Pesce

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Reverse reverses a string
func Reverse(s string) string {
	var rBuilder strings.Builder

	rns := []rune(s)
	rBuilder.Grow(len(rns))
	for i := 0; i < len(rns); i++ {
		rBuilder.WriteRune(rns[len(rns)-1-i])
	}
	return rBuilder.String()
}

var printMutex sync.Mutex

func reverseWorker(lines <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}
			rLine := Reverse(line)
			printMutex.Lock()
			fmt.Println(rLine)
			printMutex.Unlock()
		}
	}
}

var flagInput string
var flagWorkers int

func init() {
	flag.StringVar(&flagInput, "input", "", "input file")
	flag.IntVar(&flagWorkers, "workers", 4, "number of workers")
}

func main() {
	flag.Parse()
	var wg sync.WaitGroup
	lines := make(chan string, 128)
	// spawn workers
	for i := 0; i < flagWorkers; i++ {
		wg.Add(1)
		go reverseWorker(lines, &wg)
	}
	file, err := os.Open(flagInput)
	if err != nil {
		log.Fatalf("can't open(%s): %s\n", flagInput, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines)
	wg.Wait()
}
