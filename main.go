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
	flagShowOnlyResult  = flag.Bool("result", false, "不显示中间步骤，只显示解")
	flagShowStopAtFirst = flag.Bool("one", false, "找到一个解即停止")
	flagShowStat = flag.Bool("stat", false, "显示运算统计信息")
	flagShowBranch = flag.Bool("branch", false, "显示分支结构")
	flagBenchmark = flag.Bool("b", false, "(Benchmark)相当于 -result -one -stat 组合")
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
		*flagShowStat = true
	}

	puzzle := loadPuzzle()
	s, t := ParseSituation(puzzle)

	if !*flagShowOnlyResult {
		s.Show("开始", -1, -1)
	}

	startTime := time.Now()
	ctx := newSudokuContext()
	result := ctx.recurseEval(s, t, "/")
	dur := time.Since(startTime)
	if len(result) > 0 {
		fmt.Printf("\n找到了 %d 个解\n", len(result))
		for i, answer := range result {
			ShowCells(answer, fmt.Sprintf("result %d", i), -1, -1)
		}
	} else {
		s.Show("失败", -1, -1)
	}
	if *flagShowStat {
		fmt.Printf("\n总耗时：%v\n总推演次数：%d\n", dur, ctx.evalCount)
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
