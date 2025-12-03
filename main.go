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
	flagEnableBlockSlice    = flag.Bool("blockslice", true, "启用 宫区数组 排除规则")
	flagEnableExplicitPairs = flag.Bool("explicitpairs", true, "启用 显性数对 排除规则")
	flagEnableHiddenPairs   = flag.Bool("hiddenpairs", true, "启用 隐性数对 排除规则")
	flagEnableXWing         = flag.Bool("xwing", true, "启用 X-Wing 排除规则")
	flagDisableRules        = flag.Bool("norules", false, "禁用所有高级排除规则, 等同于 -blockslice=false -explicitpairs=false -hiddenpairs=false -xwing=false")
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

	if *flagDisableRules {
		*flagEnableBlockSlice = false
		*flagEnableExplicitPairs = false
		*flagEnableHiddenPairs = false
		*flagEnableXWing = false
	}

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
