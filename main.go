package main

import (
	"fmt"
)

const puzzleExpert = `
 1    5 4
 96  7   
   2   1 
      8 7
 85 6   2
  4      
 3     9 
  9 3   5
   54  6 
`

func main() {
	s := NewSituation(puzzleExpert)
	s.Show("start", -1, -1)
	result := recurseEval(s)
	if len(result) > 0 {
		for i, answer := range result {
			ShowCells(answer, fmt.Sprintf("result %d", i), -1, -1)
		}
	} else {
		s.Show("failed", -1, -1)
	}
}

func recurseEval(s *Situation) []*[9][9]int {
	consistency := eval(s)
	if !consistency {
		return nil
	}
	completed := s.Completed()
	if completed {
		cells := s.cells
		return []*[9][9]int{&cells}
	}

	result := make([]*[9][9]int, 0)
	choices := s.GuessChoices()
	if len(choices) == 0 {
		return nil
	}
	try := choices[0]

	for _, n := range try.Nums {
		s2 := s.Copy()
		s2.Set(try.Row, try.Col, n)
		s2.Show("Guess", try.Row, try.Col)
		subResult := recurseEval(s2)

		/*
		if len(subResult) == 0 {
			//exclude inconsistency guess
			fmt.Printf("inconsistency: exclude (%d,%d):%d\n",
				try.Row, try.Col, n + 1)
		} else {
			//exclude known solve
			for _, answer := range subResult {
				for r := range loop9 {
					for c := range loop9 {
						if s.cells[r][c] < 0 {
							fmt.Printf("known answer: exclude (%d,%d):%d\n",
								r, c, answer[r][c] + 1)
							s.Exclude(r, c, answer[r][c])
						}
					}
				}
			}
		}
		*/

		result = append(result, subResult...)
	}

	return result
}

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
					fmt.Printf("inconsistency locate %d in row %d\n", n+1, r)
					return false
				}
				changed = changed || changed2
				if done {
					s.Show(fmt.Sprintf("locate %d in row %d", n+1, r),
						cell.Row, cell.Col)
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
					fmt.Printf("inconsistency locate %d in column %d\n", n+1, c)
					return false
				}
				changed = changed || changed2
				if done {
					s.Show(fmt.Sprintf("locate %d in column %d", n+1, c),
						cell.Row, cell.Col)
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
						fmt.Printf("inconsistency locate %d in block (%d,%d)\n", n+1, R, C)
						return false
					}
					changed = changed || changed2
					if done {
						s.Show(fmt.Sprintf("locate %d in block (%d,%d)", n+1, R, C),
							cell.Row, cell.Col)
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
					fmt.Printf("inconsistency cell exclude (%d,%d)\n", r, c)
					return false
				}
				changed = changed || changed2
				if done {
					s.Show("cell exclude", cell.Row, cell.Col)
				}
			}
		}
		if !changed {
			break
		}
	}

	return true
}