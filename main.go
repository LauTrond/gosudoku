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
	flagBenchmark = flag.Bool("b", false, "(Benchmark)相当于 -result-only 和 -stop-at-first 组合，同时显示运算耗时。")
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
	}

	puzzle := loadPuzzle()
	s := ParseSituation(puzzle)

	if !*flagShowOnlyResult {
		s.Show("start", -1, -1)
	}

	startTime := time.Now()
	result := recurseEval(s)
	dur := time.Since(startTime)
	if len(result) > 0 {
		fmt.Printf("\n找到了 %d 个解\n", len(result))
		for i, answer := range result {
			ShowCells(answer, fmt.Sprintf("result %d", i), -1, -1)
		}
	} else {
		s.Show("failed", -1, -1)
	}
	if *flagShowOnlyResult && *flagShowStopAtFirst {
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

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回nil，表示这个局势有矛盾，不存在正确的解答。
func recurseEval(s *Situation) []*[9][9]int {
	if !eval(s) {
		return nil
	}
	completed := s.Completed()
	if completed {
		cells := s.cells
		return []*[9][9]int{&cells}
	}

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

	result := make([]*[9][9]int, 0)
	//获取所有可能的选项
	choices := s.Choices()
	if len(choices) == 0 {
		return nil
	}

	hash := func(c *GuessItem) int {
		return (c.Row * 277 + c.Col * 659) % 997
	}
	//compare 是选择算法，如果 c1 优于 c2 则返回 true
	compare := func(c1, c2 *GuessItem) bool {
		//Nums数量少的优先
		if len(c1.Nums) != len(c2.Nums) {
			return len(c1.Nums) < len(c2.Nums)
		}
		//随便一个吧
		return hash(c1) < hash(c2)
	}
	try := choices[0]
	for _, c := range choices {
		if compare(c, try) {
			try = c
		}
	}

	tryNumsForShow := make([]int, len(try.Nums))
	for i, n := range try.Nums {
		tryNumsForShow[i] = n + 1
	}

	for _, n := range try.Nums {
		s2 := s.Copy()
		s2.Set(try.Row, try.Col, n)
		if !*flagShowOnlyResult {
			s2.Show(fmt.Sprintf("在可能的选项里猜一个（全部选项：%v）", tryNumsForShow), try.Row, try.Col)
		}
		subResult := recurseEval(s2)
		result = append(result, subResult...)
		if len(result) > 0 && *flagShowStopAtFirst {
			break
		}
	}

	return result
}

// eval 开始推断局势 s，直到没有找到确定的填充选项，不确保全部完成。
// 通过 s.Completed() 可以判断是否已经填入所有数字。
// 如果返回false，表示这个局势有矛盾。
func eval(s *Situation) bool {
	for {
		changed := false
		for r := range loop9 {
			for n := range loop9 {
				ex := s.NewExcluding()
				for c := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistency, cell := ex.Apply()
				if !consistency {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：第 %d 行没有单元格可以填入 %d\n", r+1, n+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show(fmt.Sprintf("该行唯一可以填入 %d 的单元格", n+1),
							cell.Row, cell.Col)
					}
				}
			}
		}
		for c := range loop9 {
			for n := range loop9 {
				ex := s.NewExcluding()
				for r := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistency, cell := ex.Apply()
				if !consistency {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：第 %d 列没有单元格可以填入 %d\n", c+1, n+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show(fmt.Sprintf("该列唯一可以填入 %d 的单元格", n+1),
							cell.Row, cell.Col)
					}
				}
			}
		}
		for R := range loop3 {
			for C := range loop3 {
				for n := range loop9 {
					ex := s.NewExcluding()
					for rr := range loop3 {
						for cc := range loop3 {
							r := R*3 + rr
							c := C*3 + cc
							ex.Test(r, c, n)
						}
					}
					done, changed2, consistency, cell := ex.Apply()
					if !consistency {
						if !*flagShowOnlyResult {
							fmt.Printf("发生矛盾：区块(%d行,%d列)内没有单元格可以填入 %d\n", R+1, C+1, n+1)
						}
						return false
					}
					changed = changed || changed2
					if done {
						if !*flagShowOnlyResult {
							s.Show(fmt.Sprintf("区块内唯一可以填入 %d 的单元格", n+1),
								cell.Row, cell.Col)
						}
					}
				}
			}
		}
		for r := range loop9 {
			for c := range loop9 {
				ex := s.NewExcluding()
				for n := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistent, cell := ex.Apply()
				if !consistent {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：%d 行 %d 列单元格没有可以填入的数字\n", r+1, c+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show("这个单元格唯一可以填的数", cell.Row, cell.Col)
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	return true
}
