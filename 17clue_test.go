package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

//这是一个比赛：
//https://codegolf.stackexchange.com/questions/190727/the-fastest-sudoku-solver

func Test17Clue(t *testing.T) {
	*flagShowOnlyResult = true
	*flagShowStopAtFirst = true

	parallel := 8
	throttle := make(chan struct{}, parallel)
	runtime.GOMAXPROCS(parallel)
	var wg sync.WaitGroup

	raw, err := ioutil.ReadFile("assests/all_17_clue_sudokus.txt")
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(raw, []byte("\n"))

	linesNum, err := strconv.Atoi(string(lines[0]))
	if err != nil {
		t.Fatal(err)
	}

	for i := range lines[1:linesNum+1] {
		linePtr := &lines[1+i]
		throttle<-struct{}{}
		wg.Add(1)
		go func() {
			s, trg := ParseSituationFromLine(*linePtr)
			ctx := newSudokuContext()
			result := ctx.recurseEval(s, trg, "/")
			if len(result) != 1 {
				t.Fatal("unsolved:" + string(*linePtr))
			}
			s.Release()
			resultSerial := make([]byte, 81)
			for j := range resultSerial {
				resultSerial[j] = byte(result[0][j/9][j%9]) + '1'
			}
			*linePtr = append(*linePtr, ',')
			*linePtr = append(*linePtr, resultSerial...)

			wg.Done()
			<-throttle
		}()
	}
	wg.Wait()

	f, err := os.Create("assests/17clude_result.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines[:linesNum+1] {
		_, err = f.Write(line)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write([]byte{'\n'})
		if err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
}
