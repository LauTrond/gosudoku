package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
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
//本程序成绩是1秒左右，使用了多线程。无多线程成绩约3秒。

const inputFile = "assests/all_17_clue_sudokus.txt"
const outputFile = "assests/17clude_result.txt"

func Test17Clue(t *testing.T) {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	*flagShowOnlyResult = true
	*flagShowStopAtFirst = true

	parallel := 8
	throttle := make(chan struct{}, parallel)
	runtime.GOMAXPROCS(parallel)

	input, err := os.Open(inputFile); check(err)
	defer input.Close()
	br := bufio.NewReader(input)
	firstLine, err := br.ReadString('\n'); check(err)
	total, err := strconv.Atoi(strings.TrimSuffix(firstLine,"\n")); check(err)

	outputLines := make([]chan []byte, total)
	for i := 0; i < total; i++ {
		c := make(chan []byte, 1)
		outputLines[i] = c
		line, err := br.ReadBytes('\n')
		if err != io.EOF {
			check(err)
		}
		throttle<-struct{}{}
		go func() {
			defer func(){<-throttle}()

			line = bytes.TrimSuffix(line,[]byte("\n"))
			s, trg := ParseSituationFromLine(line)
			ctx := newSudokuContext()
			ctx.Run(s, trg)
			if len(ctx.results) != 1 {
				t.Error("unsolved:" + string(line))
				return
			}
			result := ctx.results[0]
			trg.Release()
			s.Release()

			outline := make([]byte, 81 + 1 + 81 + 1)
			copy(outline[0:81], line)
			outline[81] = ','
			for r := range loop9 {
				for c := range loop9 {
					outline[82 + r*9 + c] = byte(result[r][c]) + '1'
				}
			}
			outline[81 + 1 + 81] = '\n'
			c<-outline
		}()
	}

	output, err := os.Create(outputFile); check(err)
	defer output.Close()
	_, err = fmt.Fprintln(output, total); check(err)
	for _, c := range outputLines {
		_, err = output.Write(<-c); check(err)
	}
}
