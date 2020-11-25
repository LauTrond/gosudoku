package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//这是一个比赛：
//https://codegolf.stackexchange.com/questions/190727/the-fastest-sudoku-solver
//
//解49151条17线索题，需要按规定输出文件并使用MD5检验，按耗时记成绩。
//最快纪录是 Tdoku C++ 的 0.2 秒。
//本程序使用了多线程

var parallel17Clue = runtime.NumCPU()

const inputFile17Clue = "assets/17_clue.txt"
const outputFile17Clue = "output/17_clue_contest.txt"

func Test17ClueContest(t *testing.T) {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	*flagShowOnlyResult = true
	*flagStopAtFirstSolution = true

	runtime.GOMAXPROCS(parallel17Clue)
	throttle := make(chan struct{}, parallel17Clue)

	input, err := os.Open(inputFile17Clue)
	check(err)
	defer input.Close()
	br := bufio.NewReader(input)
	firstLine, err := br.ReadString('\n')
	check(err)
	total, err := strconv.Atoi(strings.TrimSuffix(firstLine, "\n"))
	check(err)

	err = os.MkdirAll(filepath.Dir(outputFile17Clue), 0755)
	check(err)
	output, err := os.Create(outputFile17Clue)
	check(err)
	defer output.Close()

	outputChannels := make(chan chan []byte, 1024)
	go func() {
		for i := 0; i < total; i++ {
			c := make(chan []byte, 1)
			outputChannels <- c
			line, err := br.ReadBytes('\n')
			if err != io.EOF {
				check(err)
			}
			throttle <- struct{}{}
			go func() {
				defer func() { <-throttle }()

				line = bytes.TrimSuffix(line, []byte("\n"))
				s, trg := ParseSituationFromLine(line)
				ctx := newSudokuContext()
				ctx.Run(s, trg)
				if len(ctx.solutions) != 1 {
					t.Error("unsolved:" + string(line))
					return
				}
				solution := ctx.solutions[0]
				s.Release()

				outline := make([]byte, 81+1+81+1)
				copy(outline[0:81], line)
				outline[81] = ','
				for r := range loop9 {
					for c := range loop9 {
						outline[82+r*9+c] = byte(solution[r][c]) + '1'
					}
				}
				outline[81+1+81] = '\n'
				c <- outline
			}()
		}
		close(outputChannels)
	}()

	_, err = fmt.Fprintln(output, total)
	check(err)
	for {
		c, ok := <-outputChannels
		if !ok {
			break
		}
		_, err = output.Write(<-c)
		check(err)
	}
}
