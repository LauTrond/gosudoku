package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func Test17Clue(t *testing.T) {
	benchmark(t, 1, "assets/17_clue.txt", "output/17_clue.txt")
}

func TestHardest1106(t *testing.T) {
	benchmark(t, 1, "assets/hardest_1106.txt", "output/hardest_1106.txt")
}

func TestHardest1905_11(t *testing.T) {
	benchmark(t, 1, "assets/hardest_1905_11.txt", "output/hardest_1905_11.txt")
}

func benchmark(t *testing.T, parallel int, inputFile, outputFile string) {
	if parallel < 1 {
		t.Fatal("parallel must >= 1")
	}

	runtime.GOMAXPROCS(parallel+1)
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	*flagShowOnlyResult = true
	*flagShowStopAtFirst = true

	input, err := os.Open(inputFile); check(err)
	defer input.Close()
	br := bufio.NewReader(input)

	err = os.MkdirAll(filepath.Dir(outputFile), 0755); check(err)
	output, err := os.Create(outputFile); check(err)
	defer output.Close()

	var mtx sync.Mutex
	puzzlesCount := 0
	guessesCount := 0
	evalCount := 0
	startTime := time.Now()

	getLine := func() ([]byte, bool) {
		for {
			line, err := br.ReadBytes('\n')
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			if len(line) == 0 {
				return nil, false
			}
			line = bytes.TrimSuffix(line, []byte("\n"))
			if len(line) != 81 {
				continue
			}
			return line, true
		}
	}

	proceed := func(line []byte) []byte {
		s, trg := ParseSituationFromLine(line)
		ctx := newSudokuContext()
		ctx.Run(s, trg)
		if len(ctx.results) != 1 {
			t.Fatal("unsolved:" + string(line))
		}
		result := ctx.results[0]
		resultBytes := make([]byte, 82)
		for r := range loop9 {
			for c := range loop9 {
				resultBytes[r*9+c] = byte('1' + result[r][c])
			}
		}
		resultBytes[81] = '\n'

		mtx.Lock()
		puzzlesCount += 1
		guessesCount += ctx.guessesCount
		evalCount += ctx.evalCount
		mtx.Unlock()
		s.Release()

		return resultBytes
	}

	if parallel == 1 {
		for {
			line, ok := getLine()
			if !ok {
				break
			}
			resultBytes := proceed(line)
			_, err = output.Write(resultBytes)
			check(err)
		}
	} else {
		outputChannels := make(chan chan []byte, 1024)
		throttle := make(chan struct{}, parallel)

		go func() {
			for {
				line, ok := getLine()
				if !ok {
					break
				}
				throttle <- struct{}{}
				outputChan := make(chan []byte, 1)
				outputChannels <- outputChan
				go func() {
					defer func() { <-throttle }()
					resultBytes := proceed(line)
					outputChan <- resultBytes
				}()
			}
			close(outputChannels)
		}()

		for {
			c,ok := <-outputChannels
			if !ok {
				break
			}
			_, err = output.Write(<-c); check(err)
		}
	}

	fmt.Printf("总耗时：%v\n", time.Since(startTime).String())
	fmt.Printf("总局数：%d\n", puzzlesCount)
	fmt.Printf("总猜次数：%d\n", guessesCount)
	fmt.Printf("总演算次数：%d\n", evalCount)
}
