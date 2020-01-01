package main

import (
	"fmt"
)

var (
	loop9 [9]bool
	loop3 [3]bool
)

const puzzleGuest = `
 1    5 4
 96  7  3
   2   1 
      8 7
 85 6   2
  4      
 3     9 
  9 3   5
   54  6 
`

const puzzleOrigin = `
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
	s := NewSituation(puzzleOrigin)
	s.show("start", -1, -1)
	for {
		found := false
		for r := range loop9 {
			for c := range loop9 {
				ex := 0
				nx := -1
				for n := range loop9 {
					if s.cellNumberEx[r][c][n] {
						ex++
					} else {
						nx = n
					}
				}
				if ex == 9 {
					panic(fmt.Errorf("unsolved at (%d,%d)",r,c))
				}
				if ex == 8 && s.cellIs[r][c] != nx {
					found = true
					s.Set(r, c, nx)
					s.show("cellNumberEx", r, c)
				}
			}
		}
		for r := range loop9 {
			for n := range loop9 {
				ex := 0
				cx := -1
				for c := range loop9 {
					if s.rowNumberColumnEx[r][n][c] {
						ex++
					} else {
						cx = c
					}
				}
				if ex == 8 && s.cellIs[r][cx] != n {
					found = true
					s.Set(r, cx, n)
					s.show("rowNumberColumnEx", r, cx)
				}
			}
		}
		for c := range loop9 {
			for n := range loop9 {
				ex := 0
				rx := -1
				for r := range loop9 {
					if s.columnNumberRowEx[c][n][r] {
						ex++
					} else {
						rx = r
					}
				}
				if ex == 8 && s.cellIs[rx][c] != n {
					found = true
					s.Set(rx, c, n)
					s.show("columnNumberRowEx", rx, c)
				}
			}
		}
		for R := range loop3 {
			for C := range loop3 {
				for n := range loop9 {
					ex := 0
					rx := -1
					cx := -1
					for r := range loop3 {
						for c := range loop3 {
							if s.blockNumberCellEx[R][C][n][r][c] {
								ex++
							} else {
								rx = r
								cx = c
							}
						}
					}
					rr := R*3 + rx
					cc := C*3 + cx
					if ex == 8 && s.cellIs[rr][cc] != n {
						found = true
						s.Set(rr, cc, n)
						s.show("blockNumberCellEx", rr, cc)
					}
				}
			}
		}
		if !found {
			break
		}
	}
}
