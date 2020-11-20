package main

import (
	"fmt"
	"sort"
	"testing"
)

const puzzle = `
.1....5.4
.96..7...
...2...1.
......8.7
.85.6...2
..4......
.3.....9.
..9.3...5
...54..6.
`

func TestBranch(t *testing.T) {
	*flagShowOnlyResult = true
	s, trg := ParseSituation(puzzle)
	newSudokuContext().logicalEval(s, trg)
	if s.Completed() {
		t.Fatalf("no branch")
	}
	s.Show("start",-1,-1)

	choices := s.Choices()
	sort.Slice(choices, func(i, j int) bool {
		return s.CompareGuestItem(choices[i], choices[j])
	})
	for d, try := range choices {
		for i, n := range try.Nums {
			s2 := s.Copy()
			trg := &Trigger{}
			s2.Set(trg, RowColNum{RowCol: try.RowCol, Num: n})
			ok := newSudokuContext().logicalEval(s2, trg)
			if !ok {
				continue
			}
			for r := range loop9 {
				for c := range loop9 {
					if s.cells[r][c] >= 0 {
						s2.cells[r][c] = -1
					}
				}
			}
			s2.Show(fmt.Sprintf("try %d choice %d: finished=%v",
				d, i, s2.Completed()), int(try.Row), int(try.Col))
		}
	}
}
