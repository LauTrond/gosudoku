package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func Test17Clue(t *testing.T) {
	RunSingleThread(t, "assets/17_clue.txt", "output/17_clue.txt")
}

func TestHardest1106(t *testing.T) {
	RunSingleThread(t, "assets/hardest_1106.txt", "output/hardest_1106.txt")
}

func TestHardest1905_11(t *testing.T) {
	RunSingleThread(t, "assets/hardest_1905_11.txt", "output/hardest_1905_11.txt")
}

func RunSingleThread(t *testing.T, inputFile, outputFile string) {
	runtime.GOMAXPROCS(2)
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

	puzzlesCount := 0
	guessesCount := 0
	evalCount := 0
	startTime := time.Now()

	for {
		line, err := br.ReadBytes('\n')
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if len(line) == 0 {
			break
		}
		line = bytes.TrimSuffix(line,[]byte("\n"))
		if len(line) != 81 {
			continue
		}
		puzzlesCount++

		s, trg := ParseSituationFromLine(line)
		ctx := newSudokuContext()
		ctx.Run(s, trg)
		if len(ctx.results) != 1 {
			t.Fatal("unsolved:" + string(line))
		} else {
			result := ctx.results[0]
			resultBytes := make([]byte, 82)
			for r := range loop9 {
				for c := range loop9 {
					resultBytes[r*9+c] = byte('1' + result[r][c])
				}
			}
			resultBytes[81] = '\n'
			_, err = output.Write(resultBytes); check(err)
		}
		guessesCount += ctx.guessesCount
		evalCount += ctx.evalCount
		s.Release()
	}

	fmt.Printf("总耗时：%v\n", time.Since(startTime).String())
	fmt.Printf("总局数：%d\n", puzzlesCount)
	fmt.Printf("总猜次数：%d\n", guessesCount)
	fmt.Printf("总演算次数：%d\n", evalCount)
}
