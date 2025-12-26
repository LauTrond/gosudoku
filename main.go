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
	flagGensApplyRules      = flag.Int("gens-apply-rules", 0, "在N代分支内使用复杂排除规则")
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

	ctx := &SudokuContext{
		ShowProcess:         *flagShowProcess,
		ShowBranch:          *flagShowBranch,
		StopAtFirstSolution: *flagStopAtFirstSolution,
		GensApplyRules:      *flagGensApplyRules,
	}
	startTime := time.Now()
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
		var sumBranches int
		for _, branches := range ctx.branchCount {
			sumBranches += branches
		}
		fmt.Printf("总耗时：%v\n", dur)
		fmt.Printf("二叉分支数：%d\n", ctx.branchCount[2])
		fmt.Printf("多叉支数：%d\n", sumBranches-ctx.branchCount[2])
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
