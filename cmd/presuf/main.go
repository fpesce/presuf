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
	"container/heap"

	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"
)

var flagFilename string
var flagCryptPerSecond int64
var flagDuration string
var flagMinPrefixLen int /* e.g. 4 */
var flagMaxPrefixLen int /* e.g. 7 */
var flagPasswordsLen int /* e.g. 10 */
var flagAlphabetSize int /* e.g. 95 */

func init() {
	flag.StringVar(&flagFilename, "input", "", "name of the sorted word file")
	flag.Int64Var(&flagCryptPerSecond, "crypt-per-sec", 20000000, "reported crypt per second (e.g. John the ripper c/s or hashcat Speed.#*)")
	flag.StringVar(&flagDuration, "duration", "168h", "time to run the cracking session on all prefixes")
	flag.IntVar(&flagMinPrefixLen, "min-prefix", 4, "minimal length of the Prefix space to search")
	flag.IntVar(&flagMaxPrefixLen, "max-prefix", 7, "maximal length of the Prefix space to search")
	flag.IntVar(&flagPasswordsLen, "pass-len", 10, "size of the passwords that will be generated (bruteforce + Prefix)")
	flag.IntVar(&flagAlphabetSize, "alphabet-size", 95, "cardinality of the alphabet (e.g. 10 for decimal digits) used in bruteforce")
}

func runeSliceIsEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// A prefixCount is a structure to keep track of number of hits that a prefix match
type prefixCount struct {
	Prefix string
	Count  int
}

// An prefixCountHeap is a min-heap of Prefix counts.
type prefixCountHeap []prefixCount

func (pch prefixCountHeap) Len() int           { return len(pch) }
func (pch prefixCountHeap) Less(i, j int) bool { return pch[i].Count < pch[j].Count }
func (pch prefixCountHeap) Swap(i, j int)      { pch[i], pch[j] = pch[j], pch[i] }

// Push adds an element to the heap
func (pch *prefixCountHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*pch = append(*pch, x.(prefixCount))
}

// Pop retrieves the minimal element of the heap
func (pch *prefixCountHeap) Pop() interface{} {
	old := *pch
	n := len(old)
	x := old[n-1]
	*pch = old[0 : n-1]
	return x
}

func main() {
	flag.Parse()

	/* Because of low number of operation (password list is sorted, so one insertion per prefix),
	   we could use a heap to keep things ordered all along.
	   We need (flagMaxPrefixLen - flagMinPrefixLen) + 1 heaps. */

	if flagFilename == "" {
		log.Fatalf("provide a -input file parameter")
	}

	/* compute the number of candidates we can try for each prefix in the offered time. */
	nbPrefixes := 1 + flagMaxPrefixLen - flagMinPrefixLen
	wholeDuration, err := time.ParseDuration(flagDuration)
	if err != nil {
		log.Fatalf("invalid duration: %s", flagDuration)
	}
	prefixDurationSeconds := wholeDuration.Seconds() / float64(nbPrefixes)
	nbCandidates := make([]int, nbPrefixes)
	nbCandidates[0] = int(prefixDurationSeconds * float64(flagCryptPerSecond) / math.Pow(float64(flagAlphabetSize), float64(flagPasswordsLen-flagMinPrefixLen)))
	fmt.Fprintf(os.Stderr, "candidates for prefix of length %d : %d\n", flagMinPrefixLen, nbCandidates[0])
	for i := 1; i < nbPrefixes; i++ {
		nbCandidates[i] = nbCandidates[i-1] * flagAlphabetSize
		fmt.Fprintf(os.Stderr, "candidates for prefix of length %d : %d\n", flagMinPrefixLen+i, nbCandidates[i])
	}
	/* N is the number of prefixes */
	/* We do N passes from longest prefixes to shortest ones, the longest will be used to then exclude shorter prefixes.
	   Exclude here means: if we select a prefix for bruteforce, we need to remove the matches (count of occurrences)
	   from longer prefixes, as we will explore that subspace. */
	file, err := os.Open(flagFilename)
	if err != nil {
		log.Fatalf("can't open file(%s):%s\n", flagFilename, err)
	}
	/* We need N hashes, N heaps */
	var prefixMaps = make([]map[string]int, nbPrefixes)
	for i := range prefixMaps {
		prefixMaps[i] = make(map[string]int, nbCandidates[i])
	}
	for currentPass := nbPrefixes - 1; currentPass >= 0; currentPass-- {
		currentPrefixHeap := make(prefixCountHeap, 0, nbCandidates[currentPass])
		heap.Init(&currentPrefixHeap)
		fmt.Fprintf(os.Stderr, "Pass %d\n", nbPrefixes-currentPass)
		_, err = file.Seek(0, os.SEEK_SET)
		if err != nil {
			log.Fatal("can't get back to the beginning of file")
		}
		scanner := bufio.NewScanner(file)
		oldWord := ""
		oldPrefixes := make([][]rune, nbPrefixes)
		for i := range oldPrefixes {
			prefixLen := i + flagMinPrefixLen
			oldPrefixes[i] = make([]rune, prefixLen)
		}
		oldPrefixesCounts := make([]int, nbPrefixes)
		for scanner.Scan() {
			wordStr := scanner.Text()
			if wordStr < oldWord {
				log.Fatalf("word list don't seem ordered: %s < %s\n", wordStr, oldWord)
			}
			oldWord = wordStr
			word := []rune(wordStr)
			for i := nbPrefixes - 1; i >= currentPass; i-- {
				prefixLen := i + flagMinPrefixLen
				if len(word) < prefixLen {
					continue
				}
				if runeSliceIsEqual(word[0:prefixLen], oldPrefixes[i]) {
					oldPrefixesCounts[i]++
				} else {
					/* We need to store the prefix in heap if we are on the corresponding currentPass */
					if i == currentPass {
						heap.Push(&currentPrefixHeap, prefixCount{
							Prefix: string(oldPrefixes[currentPass]),
							Count:  oldPrefixesCounts[currentPass],
						})
						if len(currentPrefixHeap) >= nbCandidates[i] {
							heap.Pop(&currentPrefixHeap)
						}
					} else {
						// Now if we are on a later pass than current prefix length, we use cache/map to adjust shorter prefix count
						if _, ok := prefixMaps[i][string(oldPrefixes[i])]; ok {
							for j := i - 1; j >= 0; j-- {
								oldPrefixesCounts[j] -= oldPrefixesCounts[i]
							}
						}
					}
					copy(oldPrefixes[i], word[0:prefixLen])
					oldPrefixesCounts[i] = 1
				}
			}
		}
		// Flush remaining prefix
		heap.Push(&currentPrefixHeap, prefixCount{
			Prefix: string(oldPrefixes[currentPass]),
			Count:  oldPrefixesCounts[currentPass],
		})
		if len(currentPrefixHeap) >= nbCandidates[currentPass] {
			heap.Pop(&currentPrefixHeap)
		}
		// Now cache results for that pass in the corresponding map
		for len(currentPrefixHeap) > 0 {
			prefix := heap.Pop(&currentPrefixHeap).(prefixCount)
			prefixMaps[currentPass][prefix.Prefix] = prefix.Count
		}
	}
	/* Now for the grand finale */
	for i := 0; i < nbPrefixes; i++ {
		for k, v := range prefixMaps[i] {
			/* avoid re-exploring subspace already done by shorter prefixes:
			TODO: An advanced backward and then forward propagation of the prefix in a 2nd set of passes.
			*/
			hasShortestPrefixMatched := false
			if i != 0 {
				for j := 0; j < i && !hasShortestPrefixMatched; j++ {
					prefixLen := j + flagMinPrefixLen
					if _, ok := prefixMaps[j][k[0:prefixLen]]; ok {
						hasShortestPrefixMatched = true
					}
				}
			}
			if !hasShortestPrefixMatched {
				fmt.Printf("%d:%s\n", v, k)
			}
		}
	}
}
