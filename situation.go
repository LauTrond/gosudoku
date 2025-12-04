package main

import (
	"fmt"
	"hash/crc64"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	loop9     [9]int
	loop3     [3]int
	loop3skip = [3][2]int{{1, 2}, {0, 2}, {0, 1}}
	loop9skip = [3][6]int{
		{3, 4, 5, 6, 7, 8},
		{0, 1, 2, 6, 7, 8},
		{0, 1, 2, 3, 4, 5},
	}
)

const (
	ExcludeDirectionNone  = 0
	ExcludeDirectionCell  = 1
	ExcludeDirectionRow   = 2
	ExcludeDirectionCol   = 3
	ExcludeDirectionBlock = 4
)

func RCtoBP(r, c int) (b int, p int) {
	b = r/3*3 + c/3
	p = r%3*3 + c%3
	return
}

func BPtoRC(b, p int) (r int, c int) {
	r = b/3*3 + p/3
	c = b%3*3 + p%3
	return
}

// Situation 代表9*9数独的一个局势
// 所有行、列、单元格值都是0~8，显示的时候加1处理。
type Situation struct {
	//cell[r][c] = n
	//n >=0 : 单元格(r,c) 填入了数字n
	//n == -1 : 单元格(r,c) 还没填入数字
	cells [9][9]int8

	//已填单元格总数
	setCount int

	//numSetCount[n] = x ： n 已填充 x 次
	numSetCount [9]int8
	//rowSetCount[r] = x ： r 行已填充 x 个数
	rowSetCount [9]int8
	//colSetCount[c] = x ： c 列已填充 x 个数
	colSetCount [9]int8
	//blockSetCount[b] = x ： 宫 b 已填充 x 个数
	blockSetCount [9]int8

	//cellExclude[n][r][c] = 1 ： 单元格(r,c)排除n
	cellExclude [9][9][9]int8

	//cellExcludeBits[r][c] 的每一位代表该单元格排除了哪些数字
	cellExcludeBits [9][9]int16

	//numExcludes[n] ： n 的总排除次数
	numExcludes [9]int8

	//cellNumExcludes[r][c] = x ： r 行 c 列 排除了 x 个数
	//等于 sum(cellExclude[...][r][c])
	cellNumExcludes [9][9]int8

	//rowExcludes[n][r] = x ： r 行的 n 排除了 x 个单元格
	//等于 sum(cellExclude[n][r][...])
	rowExcludes [9][9]int8

	//rowExcludeBits[n][r] 的每一位代表 r 行排除了哪些单元格
	rowExcludeBits [9][9]int16

	//rowSumExcludes[r] = x ： r 行总共排除了 x 种可能性
	//rowSumExcludes[r] = sum(rowExcludes[...][r])
	rowSumExcludes [9]int8

	//rowExcludes[n][r][C] = x ： 第 r 行 C 宫的 n 排除了 x 个单元格
	//rowExcludes[n][r][C] = sum(cellExclude[n][r][C*3..C*3+2])
	rowSliceExcludes [9][9][3]int8

	//colExcludes[n][c] = x ：c 列的 n 排除了 x 个单元格
	//colExcludes[n][c] = sum(cellExclude[n][...][c])
	colExcludes [9][9]int8

	//colExcludeBits[n][c] 的每一位代表 c 列排除了哪些单元格
	colExcludeBits [9][9]int16

	//colSumExcludes[c] = x ： c 列总共排除了 x 种可能性
	//colSumExcludes[c] = sum(colExcludes[...][c])
	colSumExcludes [9]int8

	//colSliceExcludes[n][R][c] = x ： c 列 R 宫的 n 排除了 x 个单元格
	//colSliceExcludes[n][R][c] = sum(cellExclude[n][R*3..R*3+2][c])
	colSliceExcludes [9][3][9]int8

	//blockExcludes[n][b] = x ：宫 b 的 n 排除了 x 个单元格
	blockExcludes [9][9]int8

	//blockExcludeBits[n][b] 的每一位代表 宫 b 排除了哪些单元格
	blockExcludeBits [9][9]int16

	//blockSumExcludes[b] = x ： 宫 b 总共排除了 x 种可能性
	//blockSumExcludes[b] = sum(blockExcludes[...][b])
	blockSumExcludes [9]int8

	//分支代数，每执行一次Copy就加1
	branchGeneration int
}

// 初始化一个数独谜题
// puzzle 是一个9行的字符串（前后空行会自动去除），以点和数字代表单元格
// 其中空格代表未填入的单元格
func ParseSituation(puzzle string) (*Situation, *Trigger) {
	s := NewSituation()
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	t := &Trigger{}
	for r, line := range lines {
		if r > len(s.cells) {
			panic(fmt.Errorf("row exceed"))
		}
		for c, n := range line {
			if n >= '1' && n <= '9' {
				s.Set(t, RCN(r, c, int(n-'1')))
			}
		}
	}
	return s, t
}

// 初始化一个数独谜题，不换行的81个字符
func ParseSituationFromLine(line []byte) (*Situation, *Trigger) {
	if len(line) != 81 {
		panic("invalid puzzle from line")
	}

	s := NewSituation()
	t := NewTrigger()
	for i, n := range line {
		if n >= '1' && n <= '9' {
			s.Set(t, RCN(i/9, i%9, int(n-'1')))
		}
	}
	return s, t
}

var situationPool = sync.Pool{
	New: func() interface{} {
		return new(Situation)
	},
}

var EmptySituation = func() *Situation {
	s := new(Situation)
	for r := range loop9 {
		for c := range loop9 {
			s.cells[r][c] = -1
		}
	}
	return s
}()

func NewSituation() *Situation {
	s := situationPool.Get().(*Situation)
	*s = *EmptySituation
	return s
}

func (s *Situation) Release() {
	situationPool.Put(s)
}

func (s *Situation) Copy() *Situation {
	s2 := situationPool.Get().(*Situation)
	*s2 = *s
	s2.branchGeneration++
	return s2
}

func (s *Situation) Count() int {
	return s.setCount
}

// 填数
func (s *Situation) Set(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Extract()
	if s.cells[r][c] != -1 {
		return false
	}
	s.cells[r][c] = int8(n)

	b, p := RCtoBP(r, c)
	R, C := r/3, c/3
	s.setCount++
	s.numSetCount[n]++
	s.rowSetCount[r]++
	s.colSetCount[c]++
	s.blockSetCount[b]++

	if s.cellNumExcludes[r][c] < 8 {
		for n0 := range loop9 {
			if n0 != n {
				s.exclude(t, RCN(r, c, n0), ExcludeDirectionCell)
			}
		}
	}
	if s.colExcludes[n][c] < 8 {
		for _, r0 := range loop9skip[R] {
			s.exclude(t, RCN(r0, c, n), ExcludeDirectionCol)
		}
	}
	if s.rowExcludes[n][r] < 8 {
		for _, c0 := range loop9skip[C] {
			s.exclude(t, RCN(r, c0, n), ExcludeDirectionRow)
		}
	}
	if s.blockExcludes[n][b] < 8 {
		for p0 := range loop9 {
			if p0 == p {
				continue
			}
			r0, c0 := BPtoRC(b, p0)
			s.exclude(t, RCN(r0, c0, n), ExcludeDirectionBlock)
		}
	}
	s.drainExcludeQueue(t)
	return true
}

func (s *Situation) Exclude(t *Trigger, rcn RowColNum) {
	s.exclude(t, rcn, ExcludeDirectionNone)
	s.drainExcludeQueue(t)
}

func (s *Situation) enqueueExclude(t *Trigger, rcn RowColNum, direction int) bool {
	r, c, n := rcn.Extract()
	if s.cellExclude[n][r][c] > 0 {
		return false
	}
	t.EnqueueExclude(RowColNumExclude{
		RowColNum: rcn,
		Direction: direction,
	})
	return true
}

func (s *Situation) exclude(t *Trigger, rcn RowColNum, direction int) bool {
	return s.excludeOne(t, RowColNumExclude{
		RowColNum: rcn,
		Direction: direction,
	})
}

func (s *Situation) drainExcludeQueue(t *Trigger) {
	for {
		next, ok := t.DequeueExclude()
		if !ok {
			break
		}
		s.excludeOne(t, next)
		if len(t.Conflicts) > 0 {
			break
		}
	}
}

func (s *Situation) excludeOne(t *Trigger, rcne RowColNumExclude) bool {
	r, c, n := rcne.Extract()
	if s.cellExclude[n][r][c] > 0 {
		return false
	}
	s.cellExclude[n][r][c] = 1

	b, p := RCtoBP(r, c)
	R, C := r/3, c/3
	rr, cc := r-R*3, c-C*3

	s.cellExcludeBits[r][c] |= 1 << n
	cellExcludeBits := s.cellExcludeBits[r][c]
	_ = add(&s.numExcludes[n], 1)
	cellNumExcludes := add(&s.cellNumExcludes[r][c], 1)
	rowExcludes := add(&s.rowExcludes[n][r], 1)
	s.rowExcludeBits[n][r] |= 1 << c
	rowExcludeBits := s.rowExcludeBits[n][r]
	_ = add(&s.rowSumExcludes[r], 1)
	colExcludes := add(&s.colExcludes[n][c], 1)
	s.colExcludeBits[n][c] |= 1 << r
	colExcludeBits := s.colExcludeBits[n][c]
	_ = add(&s.colSumExcludes[c], 1)
	blockExcludes := add(&s.blockExcludes[n][b], 1)
	s.blockExcludeBits[n][b] |= 1 << (rr*3 + cc)
	blockExcludeBits := s.blockExcludeBits[n][b]
	_ = add(&s.blockSumExcludes[b], 1)
	rowSliceExcludes := add(&s.rowSliceExcludes[n][r][C], 1)
	colSliceExcludes := add(&s.colSliceExcludes[n][R][c], 1)

	switch cellNumExcludes {
	case 8:
		if s.cells[r][c] >= 0 {
			break
		}
		for n0 := range loop9 {
			if s.cellExclude[n0][r][c] == 0 {
				t.Confirm(RCN(r, c, n0))
				break
			}
		}
	case 9:
		reason := ""
		if *flagShowProcess {
			reason = fmt.Sprintf("单元格(%d,%d)没有可填充数字", r+1, c+1)
		}
		t.Conflict(reason)
	}

	switch rowExcludes {
	case 8:
		for c0 := range loop9 {
			if s.cellExclude[n][r][c0] == 0 && s.cells[r][c0] < 0 {
				t.Confirm(RCN(r, c0, n))
				break
			}
		}
	case 9:
		reason := ""
		if *flagShowProcess {
			reason = fmt.Sprintf("第 %d 行没有单元格可填充 %d", r+1, n+1)
		}
		t.Conflict(reason)
	}
	switch colExcludes {
	case 8:
		for r0 := range loop9 {
			if s.cellExclude[n][r0][c] == 0 && s.cells[r0][c] < 0 {
				t.Confirm(RCN(r0, c, n))
				break
			}
		}
	case 9:
		reason := ""
		if *flagShowProcess {
			reason = fmt.Sprintf("第 %d 列没有单元格可填充 %d", c+1, n+1)
		}
		t.Conflict(reason)
	}

	switch blockExcludes {
	case 8:
	loopRow:
		for p0 := range loop9 {
			r0, c0 := BPtoRC(b, p0)
			if s.cellExclude[n][r0][c0] == 0 && s.cells[r0][c0] < 0 {
				t.Confirm(RCN(r0, c0, n))
				break loopRow
			}
		}
	case 9:
		reason := ""
		if *flagShowProcess {
			reason = fmt.Sprintf("宫(%d,%d)没有单元格可填充 %d", R+1, C+1, n+1)
		}
		t.Conflict(reason)
	}

	// 宫区数对 and 宫区三数组
	// 同一宫区内，某数字只能出现在同一行或同一列的2个或3个单元格中，那同一宫、行或列的其他单元格排除这个数字
	// https://sudoku.com/zh/shu-du-gui-ze/gong-qu-kuai-shu-dui/
	// 使用了不同的算法，但本质一样。参考 README.md
	if *flagEnableBlockSlice {
		if rcne.NotFromRow() && (rowExcludes == 6 || rowExcludes == 7) {
			for _, C0 := range loop3skip[C] {
				C1 := 3 - C - C0
				if rowSliceExcludes+s.rowSliceExcludes[n][r][C0] == 6 {
					for _, rr1 := range loop3skip[rr] {
						for cc1 := range loop3 {
							s.enqueueExclude(t, RCN(R*3+rr1, C1*3+cc1, n), ExcludeDirectionBlock)
						}
					}
				}
			}
		}

		if rcne.NotFromCol() && (colExcludes == 6 || colExcludes == 7) {
			for _, R0 := range loop3skip[R] {
				R1 := 3 - R - R0
				if colSliceExcludes+s.colSliceExcludes[n][R0][c] == 6 {
					for rr1 := range loop3 {
						for _, cc1 := range loop3skip[cc] {
							s.enqueueExclude(t, RCN(R1*3+rr1, C*3+cc1, n), ExcludeDirectionBlock)
						}
					}
				}
			}
		}

		if rcne.NotFromBlock() && (blockExcludes == 6 || blockExcludes == 7) {
			for _, rr0 := range loop3skip[rr] {
				rr1 := 3 - rr - rr0
				if rowSliceExcludes+s.rowSliceExcludes[n][R*3+rr0][C] == 6 {
					for _, c0 := range loop9skip[C] {
						s.enqueueExclude(t, RCN(R*3+rr1, c0, n), ExcludeDirectionRow)
					}
				}
			}
			for _, cc0 := range loop3skip[cc] {
				cc1 := 3 - cc - cc0
				if colSliceExcludes+s.colSliceExcludes[n][R][C*3+cc0] == 6 {
					for _, r0 := range loop9skip[R] {
						s.enqueueExclude(t, RCN(r0, C*3+cc1, n), ExcludeDirectionCol)
					}
				}
			}
		}
	}

	// 显性数对
	// 同一行、列、宫中，两个单元格只能填入同样的两个数字，可以排除其他单元格填入这两个数字
	// https://sudoku.com/zh/shu-du-gui-ze/xian-xing-shu-dui/
	if *flagEnableExplicitPairs {
		if rcne.NotFromCell() && cellNumExcludes == 7 {
			foundRow := -1
			for r0 := range loop9 {
				if r0 != r && cellExcludeBits == s.cellExcludeBits[r0][c] {
					foundRow = r0
					break
				}
			}
			if foundRow >= 0 {
				// (r,c) 和 (foundRow,c) 形成显性数对
				for n0 := range loop9 {
					if cellExcludeBits&(1<<n0) > 0 {
						continue
					}
					if s.colExcludes[n0][c] >= 7 {
						continue
					}
					for r1 := range loop9 {
						if r1 != r && r1 != foundRow {
							s.enqueueExclude(t, RCN(r1, c, n0), ExcludeDirectionCol)
						}
					}
				}
			}

			foundCol := -1
			for c0 := range loop9 {
				if c0 != c && cellExcludeBits == s.cellExcludeBits[r][c0] {
					foundCol = c0
					break
				}
			}
			if foundCol >= 0 {
				// (r,c) 和 (r,foundCol) 形成显性数对
				for n0 := range loop9 {
					if cellExcludeBits&(1<<n0) > 0 {
						continue
					}
					if s.rowExcludes[n0][r] >= 7 {
						continue
					}
					for c1 := range loop9 {
						if c1 != c && c1 != foundCol {
							s.enqueueExclude(t, RCN(r, c1, n0), ExcludeDirectionRow)
						}
					}
				}
			}

			foundPos := -1
			for p0 := range loop9 {
				if p0 == p {
					continue
				}
				r0, c0 := BPtoRC(b, p0)
				if cellExcludeBits == s.cellExcludeBits[r0][c0] {
					foundPos = p0
					break
				}
			}
			if foundPos >= 0 {
				// (b,p) 和 (b,foundPos) 形成显性数对
				for n0 := range loop9 {
					if cellExcludeBits&(1<<n0) > 0 {
						continue
					}
					if s.blockExcludes[n0][b] >= 7 {
						continue
					}
					for p1 := range loop9 {
						if p1 != p && p1 != foundPos {
							r1, c1 := BPtoRC(b, p1)
							s.enqueueExclude(t, RCN(r1, c1, n0), ExcludeDirectionBlock)
						}
					}
				}
			}
		}
	}

	// 隐性数对
	// 同一行、列、宫中，两个数字只能填入同样的两个单元格，可以排除其他数字在这两单元格的可能性
	// https://sudoku.com/zh/shu-du-gui-ze/yin-xing-shu-dui/
	if *flagEnableHiddenPairs {
		if rcne.NotFromRow() && rowExcludes == 7 {
			foundNum := -1
			for n0 := range loop9 {
				if n0 != n && rowExcludeBits == s.rowExcludeBits[n0][r] {
					foundNum = n0
					break
				}
			}
			if foundNum >= 0 {
				// n 和 foundNum 形成隐性数对
				for c0 := range loop9 {
					if rowExcludeBits&(1<<c0) > 0 {
						continue
					}
					if s.cellNumExcludes[r][c0] >= 7 {
						continue
					}
					for n1 := range loop9 {
						if n1 != n && n1 != foundNum {
							s.enqueueExclude(t, RCN(r, c0, n1), ExcludeDirectionCell)
						}
					}
				}
			}
		}
		if rcne.NotFromCol() && colExcludes == 7 {
			foundNum := -1
			for n0 := range loop9 {
				if n0 != n && colExcludeBits == s.colExcludeBits[n0][c] {
					foundNum = n0
					break
				}
			}
			if foundNum >= 0 {
				// n 和 foundNum 形成隐性数对，可以排除其他数字在该列的这两单元格的可能性
				for r0 := range loop9 {
					if colExcludeBits&(1<<r0) > 0 {
						continue
					}
					if s.cellNumExcludes[r0][c] >= 7 {
						continue
					}
					for n1 := range loop9 {
						if n1 != n && n1 != foundNum {
							s.enqueueExclude(t, RCN(r0, c, n1), ExcludeDirectionCell)
						}
					}
				}
			}
		}
		if rcne.NotFromBlock() && blockExcludes == 7 {
			foundNum := -1
			for n0 := range loop9 {
				if n0 != n && blockExcludeBits == s.blockExcludeBits[n0][b] {
					foundNum = n0
					break
				}
			}
			if foundNum >= 0 {
				// n 和 foundNum 形成隐性数对，可以排除其他数字在该宫的这两单元格的可能性
				for p0 := range loop9 {
					if blockExcludeBits&(1<<p0) > 0 {
						continue
					}
					r0, c0 := BPtoRC(b, p0)
					if s.cellNumExcludes[r0][c0] >= 7 {
						continue
					}
					for n1 := range loop9 {
						if n1 != n && n1 != foundNum {
							s.enqueueExclude(t, RCN(r0, c0, n1), ExcludeDirectionCell)
						}
					}
				}
			}
		}
	}

	// X-Wing
	// 同一数字，在两行（列）中有相同的 2 个候选单元格，可以排除其他行（列）同列（行）填入 n 的可能性
	// https://sudoku.com/zh/shu-du-gui-ze/x-yi-jie-fa/
	if *flagEnableXWing {
		if rcne.NotFromRow() && rowExcludes == 7 {
			foundRow := -1
			for r0 := range loop9 {
				if r0 != r && rowExcludeBits == s.rowExcludeBits[n][r0] {
					foundRow = r0
					break
				}
			}
			if foundRow >= 0 {
				// r 和 foundRow 形成 X-Wing
				for c0 := range loop9 {
					if rowExcludeBits&(1<<c0) > 0 {
						continue
					}
					if s.colExcludes[n][c0] >= 7 {
						continue
					}
					for r1 := range loop9 {
						if r1 != r && r1 != foundRow {
							s.enqueueExclude(t, RCN(r1, c0, n), ExcludeDirectionCol)
						}
					}
				}
			}
		}
		if rcne.NotFromCol() && colExcludes == 7 {
			foundCol := -1
			for c0 := range loop9 {
				if c0 != c && colExcludeBits == s.colExcludeBits[n][c0] {
					foundCol = c0
					break
				}
			}
			if foundCol >= 0 {
				// c 和 foundCol 形成 X-Wing，可以排除其他列的 n 在这两行的可能性
				for r0 := range loop9 {
					if colExcludeBits&(1<<r0) > 0 {
						continue
					}
					if s.rowExcludes[n][r0] >= 7 {
						continue
					}
					for c1 := range loop9 {
						if c1 != c && c1 != foundCol {
							s.enqueueExclude(t, RCN(r0, c1, n), ExcludeDirectionRow)
						}
					}
				}
			}
		}
	}

	return true
}

func (s *Situation) Show(title string, r, c int) {
	lines := strings.Split(title, "\n")
	lines[0] = fmt.Sprintf("<%02d> %s", s.setCount, lines[0])
	for i := 1; i < len(lines); i++ {
		lines[i] = "     " + lines[i]
	}
	title = strings.Join(lines, "\n")
	ShowCells(&s.cells, title, r, c)
}

func (s *Situation) RowColHash(rc RowCol) int {
	return (rc.Row*317 + rc.Col*659 + s.setCount*531) % 997
}

func (s *Situation) ChooseGuessingCell1() []RowColNum {
	type Candidate struct {
		RowCol
		// numCd, rowCd, colCd, blockCd int
		Score int64
	}
	selected := Candidate{
		RowCol: RowCol{-1, -1},
		Score:  1 << 30,
	}

	isBetter := func(candidate Candidate) bool {
		if candidate.Score != selected.Score {
			return candidate.Score < selected.Score
		}
		return s.RowColHash(candidate.RowCol) < s.RowColHash(selected.RowCol)
	}
	for r := range loop9 {
		for c := range loop9 {
			b, _ := RCtoBP(r, c)
			if s.cellNumExcludes[r][c] >= 8 {
				continue
			}
			exNum := int64(9 - s.cellNumExcludes[r][c])
			exRow := int64(9 - s.rowSumExcludes[r])
			exCol := int64(9 - s.colSumExcludes[c])
			exBlock := int64(9 - s.blockSumExcludes[b])
			// setRow := int64(9 - s.rowSetCount[r])
			// setCol := int64(9 - s.colSetCount[c])
			// setBlock := int64(9 - s.blockSetCount[r/3][c/3])

			candidate := Candidate{
				RowCol: RowCol{r, c},
				Score:  1,
			}
			// candidate.Score = (exNum << 20) - exRow - exCol - exBlock //72306689
			candidate.Score = (exNum << 20) - exRow*exCol*exBlock //71240920 best
			// candidate.Score = (exNum << 20) - setRow - setCol - setBlock //75853735
			// candidate.Score = (exNum << 20) - setRow*setCol*setBlock //78121998

			if isBetter(candidate) {
				selected = candidate
			}
		}
	}
	if selected.Row == -1 || selected.Col == -1 {
		return nil
	}
	result := make([]RowColNum, 0, 9-int(s.cellNumExcludes[selected.Row][selected.Col]))
	for n := range loop9 {
		if s.cellExclude[n][selected.Row][selected.Col] == 0 {
			result = append(result, RCN(selected.Row, selected.Col, n))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return s.CompareNumInCell(selected.Row, selected.Col,
			int(result[i].Num), int(result[j].Num))
	})
	return result
}

func (s *Situation) ChooseGuessingCell2() []RowColNum {
	type Candidate struct {
		RowColNum
		Score int64
	}
	selected := Candidate{
		RowColNum: RCN(-1, -1, -1),
		Score:     1 << 30,
	}
	isBetter := func(candidate Candidate) bool {
		if candidate.Score != selected.Score {
			return candidate.Score < selected.Score
		}
		return s.RowColHash(candidate.RowCol) < s.RowColHash(selected.RowCol)
	}
	for r := range loop9 {
		for c := range loop9 {
			b, _ := RCtoBP(r, c)
			if s.cellNumExcludes[r][c] >= 8 {
				continue
			}
			exNum := int64(9 - s.cellNumExcludes[r][c])
			setRow := int64(9 - s.rowSetCount[r])
			setCol := int64(9 - s.colSetCount[c])
			setBlock := int64(9 - s.blockSetCount[b])
			preScore := (exNum << 20) - setRow - setCol - setBlock
			for n := range loop9 {
				if s.cellExclude[n][r][c] > 0 {
					continue
				}
				//exRow := int64(9 - s.rowExcludes[n][r])
				//exCol := int64(9 - s.colExcludes[n][c])
				//exBlock := int64(9 - s.blockExcludes[n][r/3][c/3])
				setNum := int64(9 - s.numSetCount[n])
				candidate := Candidate{
					RowColNum: RCN(r, c, n),
					Score:     preScore - setNum,
				}
				// 分支选择算法
				// candidate.Score = exNum //80950206
				// candidate.Score = (exNum << 20) - setNum*setRow*setCol*setBlock //72411336
				// candidate.Score = (exNum << 20) - setNum - setRow - setCol - setBlock //69618934 best
				// candidate.Score = (exNum << 20) - exRow*exCol*exBlock  //78630798
				// candidate.Score = (exNum << 20) - exRow - exCol - exBlock //76996398
				// candidate.Score = (exNum << 20) - exRow*exCol*exBlock*setNum*setRow*setCol*setBlock

				if isBetter(candidate) {
					selected = candidate
				}
			}
		}
	}
	if selected.Row == -1 {
		return nil
	}
	return []RowColNum{selected.RowColNum}
}

// 选择哪个号码开始猜测，返回true表示n1比较好
func (s *Situation) CompareNumInCell(r, c, n1, n2 int) bool {
	//我也不知道为啥这个指标会有效，只是测试结果表明，这样蒙对的概率更高
	score1 := int(s.numExcludes[n1])
	score2 := int(s.numExcludes[n2])
	if score1 != score2 {
		return score1 < score2
	}

	base := r*61 + c*67 + s.setCount*71
	return (base*n1)%41 < (base*n2)%41
}

func ShowCells(cells *[9][9]int8, title string, r, c int) {
	fmt.Println("=============================")
	fmt.Println(title)
	for r1 := range loop9 {
		for c1 := range loop9 {
			s := " "
			if n1 := cells[r1][c1]; n1 >= 0 {
				s = strconv.Itoa(int(n1 + 1))
			}
			if r1 == r && c1 == c {
				fmt.Printf("[%s]", s)
			} else {
				fmt.Printf(" %s ", s)
			}
			if c1 == 2 || c1 == 5 {
				fmt.Printf("|")
			}
		}
		fmt.Println()
		if r1 == 2 || r1 == 5 {
			fmt.Println("-----------------------------")
		}
	}
}

func (s *Situation) Hash() uint64 {
	raw := make([]byte, 9*9)
	for r := range loop9 {
		for c := range loop9 {
			raw[r*9+c] = byte(s.cells[r][c] + 1)
		}
	}

	return crc64.Checksum(raw, crc64.MakeTable(crc64.ISO))
}

func (s *Situation) Completed() bool {
	return s.setCount == 9*9
}

type RowCol struct {
	Row, Col int
}

func (rc RowCol) Block() RowCol {
	return RowCol{
		rc.Row / 3,
		rc.Col / 3,
	}
}

func (rc RowCol) LeftTop() RowCol {
	return RowCol{
		rc.Row * 3,
		rc.Col * 3,
	}
}

func (rc RowCol) Add(r, c int) RowCol {
	return RowCol{
		rc.Row + int(r),
		rc.Col + int(c),
	}
}

type RowColNum struct {
	RowCol
	Num int
}

func RCN(r, c, n int) RowColNum {
	return RowColNum{
		RowCol: RowCol{
			Row: int(r),
			Col: int(c),
		},
		Num: int(n),
	}
}

func (rcn RowColNum) Extract() (r, c, n int) {
	return int(rcn.Row), int(rcn.Col), int(rcn.Num)
}

type RowColNumExclude struct {
	RowColNum
	Direction int
}

func (rcne RowColNumExclude) NotFromCell() bool {
	return rcne.Direction != ExcludeDirectionCell
}

func (rcne RowColNumExclude) NotFromRow() bool {
	return rcne.Direction != ExcludeDirectionRow
}

func (rcne RowColNumExclude) NotFromCol() bool {
	return rcne.Direction != ExcludeDirectionCol
}

func (rcne RowColNumExclude) NotFromBlock() bool {
	return rcne.Direction != ExcludeDirectionBlock
}

var triggerPool = sync.Pool{
	New: func() interface{} {
		return &Trigger{
			Confirms: make([]RowColNum, 0, 20),
			excludes: make([]RowColNumExclude, 100),
		}
	},
}

type Trigger struct {
	Confirms    []RowColNum
	excludes    []RowColNumExclude
	excludeHead int
	excludeTail int
	Conflicts   []string
}

func NewTrigger() *Trigger {
	t := triggerPool.Get().(*Trigger)
	t.Init()
	return t
}

func (t *Trigger) Init() {
	t.Confirms = t.Confirms[:0]
	t.excludeHead = 0
	t.excludeTail = 0
	t.Conflicts = t.Conflicts[:0]
}

func (t *Trigger) Release() {
	triggerPool.Put(t)
}

func (t *Trigger) Confirm(rcn RowColNum) {
	t.Confirms = append(t.Confirms, rcn)
}

func (t *Trigger) EnqueueExclude(rcne RowColNumExclude) {
	// for i := t.excludeHead; i < t.excludeTail; i++ {
	// 	compare := &t.excludes[i%len(t.excludes)]
	// 	if compare.RowColNum == rcne.RowColNum {
	// 		compare.Direction = ExcludeDirectionNone
	// 		return
	// 	}
	// }
	if t.excludeTail-t.excludeHead >= len(t.excludes) {
		newSize := max(len(t.excludes)*2, 100)
		newExcludes := make([]RowColNumExclude, newSize)
		for i := t.excludeHead; i < t.excludeTail; i++ {
			newExcludes[i-t.excludeHead] = t.excludes[i%len(t.excludes)]
		}
		t.excludes = newExcludes
		t.excludeTail = t.excludeTail - t.excludeHead
		t.excludeHead = 0
	}
	t.excludes[t.excludeTail%len(t.excludes)] = rcne
	t.excludeTail++
}

func (t *Trigger) DequeueExclude() (RowColNumExclude, bool) {
	if t.excludeTail-t.excludeHead > 0 {
		result := t.excludes[t.excludeHead%len(t.excludes)]
		t.excludeHead++
		return result, true
	} else {
		return RowColNumExclude{}, false
	}
}

func (t *Trigger) Conflict(msg string) {
	t.Conflicts = append(t.Conflicts, msg)
}

func (t *Trigger) Copy() *Trigger {
	t2 := NewTrigger()
	t2.Confirms = append(t2.Confirms, t.Confirms...)
	t2.Conflicts = append(t2.Conflicts, t.Conflicts...)
	return t2
}

type GuessItem struct {
	RowCol
	Nums []int8
}

func add(p *int8, n int8) int8 {
	*p += n
	return *p
}
