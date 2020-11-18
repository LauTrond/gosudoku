package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

var (
	flagShowOnlyResult  = flag.Bool("result-only", false, "不显示中间步骤，只显示解")
	flagShowStopAtFirst = flag.Bool("stop-at-first", false, "找到一个解即停止")
	flagShowDuration = flag.Bool("show-duration", false, "显示运算耗时")
	flagBenchmark = flag.Bool("b", false, "(Benchmark)相当于 -result-only -stop-at-first -show-duration 组合")
)

const MsgUsage = `使用方法：

sudoku <file> 从文件加载谜题
sudoku        从标准输入获取谜题

`

func main() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(os.Stderr, MsgUsage)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()
	if *flagBenchmark {
		*flagShowOnlyResult = true
		*flagShowStopAtFirst = true
		*flagShowDuration = true
	}

	puzzle := loadPuzzle()
	s := ParseSituation(puzzle)

	if !*flagShowOnlyResult {
		s.Show("开始", -1, -1)
	}

	startTime := time.Now()
	result := newSudokuContext().recurseEval(s)
	dur := time.Since(startTime)
	if len(result) > 0 {
		fmt.Printf("\n找到了 %d 个解\n", len(result))
		for i, answer := range result {
			ShowCells(answer, fmt.Sprintf("result %d", i), -1, -1)
		}
	} else {
		s.Show("失败", -1, -1)
	}
	if *flagShowDuration {
		fmt.Printf("\n总耗时：%v\n", dur)
	}
}

func loadPuzzle() string {
	input := io.Reader(os.Stdin)
	if flag.Arg(0) != "" {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		input = f
	}

	raw, err := ioutil.ReadAll(io.LimitReader(input, 1024))
	if err != nil {
		panic(err)
	}

	return string(raw)
}
