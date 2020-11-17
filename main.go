package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
)

var (
	flagShowOnlyResult = flag.Bool("result-only", false, "不显示中间步骤，只显示答案")
	flagShowStopAtFirst = flag.Bool("stop-at-first", false, "找到一个答案即停止")
)

func main() {
	flag.Parse()
	puzzle := loadPuzzle()
	s := ParseSituation(puzzle)

	if !*flagShowOnlyResult {
		s.Show("start", -1, -1)
	}
	result := recurseEval(s)
	if len(result) > 0 {
		fmt.Printf("\n找到了 %d 个答案\n", len(result))
		for i, answer := range result {
			ShowCells(answer, fmt.Sprintf("result %d", i), -1, -1)
		}
	} else {
		s.Show("failed", -1, -1)
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

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后猜一个。
	result := make([]*[9][9]int, 0)
	choices := s.Choices()
	if len(choices) == 0 {
		return nil
	}
	sort.Slice(choices, func(i, j int) bool {
		return len(choices[i].Nums) < len(choices[j].Nums)
	})
	try := choices[0]

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
							r := R * 3 + rr
							c := C * 3 + cc
							ex.Test(r, c, n)
						}
					}
					done, changed2, consistency, cell := ex.Apply()
					if !consistency {
						if !*flagShowOnlyResult {
							fmt.Printf("发生矛盾：区块内没有单元格可以填入 %d\n", n+1)
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
