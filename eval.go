package main

import "fmt"

type SudokuContext struct {
	//triedSituation map[uint64]bool
}

func newSudokuContext() *SudokuContext {
	return &SudokuContext{
		//triedSituation: map[uint64]bool{},
	}
}

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回nil，表示这个局势有矛盾，不存在正确的解答。
func (ctx *SudokuContext) recurseEval(s *Situation, t *Trigger) []*[9][9]int {
	//if h := s.Hash(); ctx.triedSituation[h] {
	//	return nil
	//} else {
	//	ctx.triedSituation[h] = true
	//}

	if !ctx.logicalEval(s, t) {
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

	try := choices[0]
	for _, c := range choices {
		if compareGuestItem(c, try) {
			try = c
		}
	}

	for _, n := range try.Nums {
		s2 := s.Copy()
		t = &Trigger{}
		s2.Set(t, RCN(try.Row, try.Col, n))
		if !*flagShowOnlyResult {
			s2.Show("在可能的选项里猜一个", try.Row, try.Col)
		}
		subResult := ctx.recurseEval(s2, t)
		result = append(result, subResult...)
		if len(result) > 0 && *flagShowStopAtFirst {
			break
		}
	}

	return result
}

// logicalEval 开始推断局势 s，直到没有找到确定的填充选项，不确保全部完成。
// 如果返回false，表示这个局势有矛盾。
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) bool {
	checkConflicts := func(t1 *Trigger) bool {
		if len(t1.Conflicts) > 0 {
			if !*flagShowOnlyResult {
				fmt.Println("发生矛盾：")
				for _, msg := range t1.Conflicts {
					fmt.Println(msg)
				}
			}
			return false
		}
		return true
	}
	if !checkConflicts(t) {
		return false
	}
	for len(t.Confirms) > 0 || len(t.Conflicts) > 0 {
		next := &Trigger{}
		for _, rcn := range t.Confirms {
			if s.Set(next, rcn) && !*flagShowOnlyResult {
				s.Show("", rcn.Row, rcn.Col)
			}
			if !checkConflicts(next) {
				return false
			}
		}
		if len(next.Confirms) == 0 {
			s.ExcludeByRules(next)
		}
		t = next
	}
	return true
}
