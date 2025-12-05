package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

var (
	flagShowProcess         = flag.Bool("process", false, "显示中间计算步骤")
	flagStopAtFirstSolution = flag.Bool("one", false, "找到一个解即停止")
	flagShowStat            = flag.Bool("stat", false, "显示运算统计信息")
	flagShowBranch          = flag.Bool("branch", false, "显示分支结构")
	flagComplexGen          = flag.Int("complex-gen", 6, "在指定分支次数前使用高级排除规则")
	flagComplexCell         = flag.Int("complex-cell", 7, "复杂规则应用范围，有效范围5-7")
)

const MsgUsage = `使用方法：

gosudoku <file> 从文件加载谜题
gosudoku        从标准输入获取谜题

`

func main() {
	flag.CommandLine.Usage = func() {
		fmt.Fprint(os.Stderr, MsgUsage)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()

	puzzle := loadPuzzle()
	s, t := ParseSituation(puzzle)

	if *flagShowProcess {
		s.Show("开始", -1, -1)
	}

	startTime := time.Now()
	ctx := newSudokuContext()
	count := ctx.Run(s, t)
	dur := time.Since(startTime)
	if count > 0 {
		fmt.Printf("\n找到了 %d 个解\n", count)
		for i, answer := range ctx.solutions {
			ShowCells(answer, fmt.Sprintf("解 %d", i+1), -1, -1)
		}
	} else {
		s.Show("失败", -1, -1)
	}
	if *flagShowStat {
		fmt.Printf("总耗时：%v\n", dur)
		fmt.Printf("总分支数：%d\n", ctx.guessesCount)
		fmt.Printf("总演算次数 %d\n", ctx.evalCount)
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

	raw, err := io.ReadAll(io.LimitReader(input, 1024))
	if err != nil {
		panic(err)
	}

	return string(raw)
}
