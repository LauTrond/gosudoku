package main

import (
	"fmt"
	"os"
	"testing"
)

func TestExcludeRule(test *testing.T) {
	puzzle, err := os.ReadFile("puzzles/hard-02.txt")
	check(err)
	s, t := ParseSituation(string(puzzle))
	s.Show("初始", -1, -1)

	ctx := NewSudokuContext()
	count := ctx.recurseEval(DuplicateSituation(s), NewTrigger(), fmt.Sprintf("<%d>", s.Count()))
	if count != 1 {
		test.Fatalf("非唯一解：%d", count)
	}
	solution := ctx.solutions[0]
	ShowCells(solution, "解", -1, -1)

	ctx = NewSudokuContext()
	ctx.debugAnswer = solution
	ctx.ShowBranch = true
	ctx.GensApplyRules = 100
	ctx.recurseEval(s, t, "")
}
