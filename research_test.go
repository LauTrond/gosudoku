package main

import (
	"fmt"
	"os"
	"testing"
)

func TestBranch(t *testing.T) {
	puzzle, err := os.ReadFile("puzzles/hard-02.txt")
	check(err)
	s, tgr := ParseSituation(string(puzzle))
	logicalEval(s, tgr)
	s.Show("初始", -1, -1)

	ctx := newSudokuContext()
	count := ctx.recurseEval(DuplicateSituation(s), NewTrigger(), fmt.Sprintf("<%d>", s.Count()))
	if count != 1 {
		t.Fatalf("非唯一解：%d", count)
	}
	ShowCells(ctx.solutions[0], "解", -1, -1)
}

func logicalEval(s *Situation, t *Trigger) bool {
	for {
		rcn, ok := t.GetConfirm()
		if !ok {
			break
		}
		if s.Set(t, rcn) {
			if len(t.Conflicts) > 0 {
				return false
			}
		}
	}
	return true
}
