package main

import (
	"fmt"
	"strings"
)

type Situation struct {
	//rowNumberColumnEx[r][n][c] = true
	//r行的n不可能存在于c列
	rowNumberColumnEx [9][9][9]bool

	//columnNumberRowEx[c][n][r] = true
	//c列的n不可能存在于r行
	columnNumberRowEx [9][9][9]bool

	//blockNumberCellEx[R][C][n][r][c] = true
	//块(R,C)的n不可能存在于(r,c)
	blockNumberCellEx [3][3][9][3][3]bool

	//cellNumberEx[r][c][n] = true
	//r行c列不可能是n
	cellNumberEx [9][9][9]bool

	//cellNumber[r][c] = n
	//r行c列是n
	cellIs [9][9]int
}

func NewSituation(puzzle string) *Situation {
	var s Situation
	for r := range loop9 {
		for c := range loop9 {
			s.cellIs[r][c] = -1
		}
	}
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	for r, line := range lines {
		for c, n := range line {
			if n >= '1' && n <= '9' {
				s.Set(r, c, int(n-'1'))
			}
		}
	}
	return &s
}

func (s *Situation) Copy() *Situation {
	s2 := *s
	return &s2
}

//设置r行c列是n
func (s *Situation) Set(r, c, n int) {
	s.cellIs[r][c]=n
	R := r/3
	C := c/3

	for n0 := range loop9 {
		if n0 != n {
			s.Exclude(r, c, n0)
		}
	}
	for r0 := range loop9 {
		if r0 != r {
			s.Exclude(r0, c, n)
		}
	}
	for c0 := range loop9 {
		if c0 != c {
			s.Exclude(r, c0, n)
		}
	}

	for r0 := range loop3 {
		for c0 := range loop3 {
			rr := R * 3 + r0
			cc := C * 3 + c0
			if rr != r || cc != c {
				s.Exclude(rr, cc, n)
			}
		}
	}
}

func (s *Situation) Exclude(r, c, n int) {
	R := r/3
	C := c/3

	s.cellNumberEx[r][c][n] = true
	s.rowNumberColumnEx[r][n][c] = true
	s.columnNumberRowEx[c][n][r] = true
	s.blockNumberCellEx[R][C][n][r-R*3][c-C*3] = true
}

func (s *Situation) show(title string, r, c int) {
	fmt.Println("====================================")
	fmt.Println(title)
	for r1 := range loop9 {
		for c1 := range loop9 {
			if n1 := s.cellIs[r1][c1]; n1 >= 0 {
				if r1 == r && c1 == c {
					fmt.Printf("[%d] ", n1+1)
				} else {
					fmt.Printf(" %d  ", n1+1)
				}
			} else {
				fmt.Printf("    ")
			}
			if c1 == 2 || c1 == 5 {
				fmt.Printf("|")
			}
		}
		fmt.Println()
		if r1 == 2 || r1 == 5 {
			fmt.Println("------------------------------------")
		}
	}
}
