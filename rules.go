package main

import "fmt"

/*

=== 互斥组 ===

同一行、或同一列、或同一区块。

=== 占位排除法 ===

虽然有多个可能填充选项，但这些选项是同一个数N，而且都在相同的互斥组中。
那么同一互斥组的其他单元格可以排除。

例如：

     1      |            | 5   #   4
     9   6  |         7  | #   #   #
            | 2       *  |     1
------------------------------------
            |            | 8       7
     8   5  |     6      |         2
         4  |            |
------------------------------------
     3      |            |     9
         9  |     3      |         5
            | 5   4      |     6

留意右上区块，根据其他行列排除法，标记"#"的单元格都不能是 6，所以 6 必定在同一区块剩余的两个单元格中。
刚好这两个选项同在另外一个互斥体（第 3 行），所以第 3 行其他单元格都不能是 6。
标记 "*" 的单元格因此排除6，你没有其他简单的方法可以作出这个断言。

*/

func (s *Situation) ApplyRuleMultiStandCol(t *Trigger) {
	for n := range loop9 {
		var rectColExcludes [3][9]int
		for r := range loop9 {
			for c := range loop9 {
				rectColExcludes[r/3][c] += s.cellExclude[r][c][n]
			}
		}
		for R := range loop3 {
			for C := range loop3 {
				for c := range loop3 {
					sumBlockExcludes := 0
					for _, i := range loop3skip[c] {
						sumBlockExcludes += rectColExcludes[R][C*3+i]
					}
					if sumBlockExcludes == 6 {
						for _, r0 := range loop9skip[R] {
							s.Exclude(t, RCN(r0, C*3+c, n))
						}
					}
					sumColExcludes := 0
					for _, i := range loop3skip[R] {
						sumColExcludes += rectColExcludes[i][C*3+c]
					}
					if sumColExcludes == 6 {
						for r0 := range loop3 {
							for _, c0 := range loop3skip[c] {
								s.Exclude(t, RCN(R*3+r0, C*3+c0, n))
							}
						}
					}
				}
			}
		}
	}
}

func (s *Situation) ApplyRuleMultiStandRow(t *Trigger) {
	for n := range loop9 {
		var rectRowExcludes [9][3]int
		for r := range loop9 {
			for c := range loop9 {
				rectRowExcludes[r][c/3] += s.cellExclude[r][c][n]
			}
		}
		for R := range loop3 {
			for C := range loop3 {
				for r := range loop3 {
					sumBlockExcludes := 0
					for _, i := range loop3skip[r] {
						sumBlockExcludes += rectRowExcludes[R*3+i][C]
					}
					if sumBlockExcludes == 6 {
						for _, c0 := range loop9skip[C] {
							s.Exclude(t, RCN(R*3+r, c0, n))
						}
					}

					sumRowExcludes := 0
					for _, i := range loop3skip[C] {
						sumRowExcludes += rectRowExcludes[R*3+r][i]
					}
					if sumRowExcludes == 6 {
						for _, r0 := range loop3skip[r] {
							for c0 := range loop3 {
								s.Exclude(t, RCN(R*3+r0, C*3+c0, n))
							}
						}
					}
				}
			}
		}
	}
}

func (s *Situation) ApplyRuleXWingRow(t *Trigger) {
	for n := range loop9 {
		buckets := make(map[int][]int)
		for r := range loop9 {
			if s.rowExcludes[r][n] == 7 {
				rowHash := 0
				for c := range loop9 {
					rowHash += s.cellExclude[r][c][n] << c
				}
				buckets[rowHash] = append(buckets[rowHash], r)
			}
		}
		for _, rows := range buckets {
			if len(rows) != 2 {
				continue
			}
			r0, r1 := rows[0], rows[1]
			for c := range loop9 {
				if s.cellExclude[r0][c][n] == 0 {
					for rr := range loop9 {
						if rr == r0 || rr == r1 {
							continue
						}
						ok := s.Exclude(t, RCN(rr, c, n))
						if ok && !*flagShowOnlyResult {
							fmt.Printf("XWing行排除法: 根据第 %d 行和第 %d 行 %d 的位置，单元格(%d,%d)排除 %d\n",
								r0+1, r1+1, n+1, rr+1, c+1, n+1)
						}
					}
				}
			}
		}
	}
}

func (s *Situation) ApplyRuleXWingCol(t *Trigger) {
	for n := range loop9 {
		buckets := make(map[int][]int)
		for c := range loop9 {
			if s.colExcludes[c][n] == 7 {
				colHash := 0
				for r := range loop9 {
					colHash += s.cellExclude[r][c][n] << r
				}
				buckets[colHash] = append(buckets[colHash], c)
			}
		}
		for _, cols := range buckets {
			if len(cols) != 2 {
				continue
			}
			c0, c1 := cols[0], cols[1]
			for r := range loop9 {
				if s.cellExclude[r][c0][n] == 0 {
					for cc := range loop9 {
						if cc == c0 || cc == c1 {
							continue
						}
						ok := s.Exclude(t, RCN(r, cc, n))
						if ok && !*flagShowOnlyResult {
							fmt.Printf("XWing列排除法: 根据第 %d 列和第 %d 列 %d 的位置，单元格(%d,%d)排除 %d\n",
								c0+1, c1+1, n+1, r+1, cc+1, n+1)
						}
					}
				}
			}
		}
	}
}
