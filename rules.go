package main

func (s *Situation) ApplyExcludeRules(t *Trigger) (changed int) {
	// changed += s.applyDimVariantRule(t, s.getMaskNumRow, s.getMaskNumCol, NRC)
	// changed += s.applyDimVariantRule(t, s.getMaskNumCol, s.getMaskNumRow, NCR)
	// changed += s.applyDimVariantRule(t, s.getMaskRowCol, s.getMaskRowNum, RCN)
	// changed += s.applyDimVariantRule(t, s.getMaskRowNum, s.getMaskRowCol, RNC)
	// changed += s.applyDimVariantRule(t, s.getMaskColNum, s.getMaskColRow, CNR)
	// changed += s.applyDimVariantRule(t, s.getMaskColRow, s.getMaskColNum, CRN)
	// changed += s.applyDimVariantRule(t, s.getMaskBlockNum, s.getMaskBlockPos, BNPtoRCN)
	// changed += s.applyDimVariantRule(t, s.getMaskBlockPos, s.getMaskBlockNum, BPNtoRCN)
	changed += s.applyBlockExcludeRules(t)
	changed += s.applyChainExcludeRule(t)
	return
}

func (s *Situation) getMaskNumRow(n, r int8) *int16 {
	return &s.rowExcludeMask[n][r]
}

func (s *Situation) getMaskRowNum(r, n int8) *int16 {
	return &s.rowExcludeMask[n][r]
}

func (s *Situation) getMaskNumCol(n, c int8) *int16 {
	return &s.colExcludeMask[n][c]
}

func (s *Situation) getMaskColNum(c, n int8) *int16 {
	return &s.colExcludeMask[n][c]
}

func (s *Situation) getMaskRowCol(r, c int8) *int16 {
	return &s.numExcludeMask[r][c]
}

func (s *Situation) getMaskColRow(c, r int8) *int16 {
	return &s.numExcludeMask[r][c]
}

func (s *Situation) getMaskBlockNum(b, n int8) *int16 {
	return &s.blockExcludeMask[n][b]
}

func (s *Situation) getMaskBlockPos(b, p int8) *int16 {
	r, c := rcbp(b, p)
	return &s.numExcludeMask[r][c]
}

type getMaskFunc func(x, y int8) *int16
type getRowColNumFunc func(x, y, z int8) RowColNum

func (s *Situation) applyDimVariantRule(t *Trigger, getMaskDim1Dim2, getMaskDim1Dim3 getMaskFunc, getRCN getRowColNumFunc) (changed int) {
	for _dim1 := range loop9 {
		dim1 := int8(_dim1)
		var checkMap [1 << 9]int8
		for _dim2 := range loop9 {
			dim2b := int8(_dim2)
			dim3mask := *getMaskDim1Dim2(dim1, dim2b)
			if countTrueBits(dim3mask) != 7 {
				continue
			}
			if checkMap[dim3mask] == 0 {
				checkMap[dim3mask] = dim2b + 1
				continue
			}
			dim2a := checkMap[dim3mask] - 1

			dim2skipMask := skip9mask[dim2a] & skip9mask[dim2b]
			for _dim3 := range loop9 {
				dim3 := int8(_dim3)
				if dim3mask&(1<<dim3) > 0 {
					continue
				}
				dim2mask := *getMaskDim1Dim3(dim1, dim3)
				if dim2mask == dim2mask|dim2skipMask {
					continue
				}
				for _dim2c := range loop9 {
					dim2c := int8(_dim2c)
					if dim2c == dim2a || dim2c == dim2b {
						continue
					}
					changed += s.excludeOne(t, getRCN(dim1, dim2c, dim3))
				}
				if len(t.Conflicts) > 0 {
					return
				}
			}
		}

	}
	return
}

func (s *Situation) applyBlockExcludeRules(t *Trigger) (changed int) {
	for _n := range loop9 {
		n := int8(_n)
		for _r := range loop9 {
			r := int8(_r)
			rr := r % 3
			for _C := range loop3 {
				C := int8(_C)
				b := r/3*3 + C
				if s.rowExcludeMask[n][r]&skip3mask[C] == skip3mask[C] && s.blockExcludeMask[n][b]&skip3mask[rr] != skip3mask[rr] {
					for _, p := range loop9skip3[rr] {
						changed += s.excludeOne(t, BPNtoRCN(b, p, n))
					}
				}
				if s.blockExcludeMask[n][b]&skip3mask[rr] == skip3mask[rr] && s.rowExcludeMask[n][r]&skip3mask[C] != skip3mask[C] {
					for _, c := range loop9skip3[C] {
						changed += s.excludeOne(t, RCN(r, c, n))
					}
				}
			}
		}
		for _c := range loop9 {
			c := int8(_c)
			cc := c % 3
			for _R := range loop3 {
				R := int8(_R)
				b := c/3 + R*3
				if s.colExcludeMask[n][c]&skip3mask[R] == skip3mask[R] && s.blockExcludeMask[n][b]&skip3maskCol[cc] != skip3maskCol[cc] {
					for _, p := range loop9skip3col[cc] {
						changed += s.excludeOne(t, BPNtoRCN(b, p, n))
					}
				}
				if s.blockExcludeMask[n][b]&skip3maskCol[cc] == skip3maskCol[cc] && s.colExcludeMask[n][c]&skip3mask[R] != skip3mask[R] {
					for _, r := range loop9skip3[R] {
						changed += s.excludeOne(t, RCN(r, c, n))
					}
				}
			}
		}
	}
	return
}

// 强弱链排除规则
func (s *Situation) applyChainExcludeRule(t *Trigger) (changed int) {
	var chain Chain
	s.setStrongLinks(&chain)
	chain.Prune()
	var chainWalker = ChainWalker{
		Chain: &chain,
	}
	chainWalker.WalkAll()
	return s.applyWeakLinkExclude(t, &chainWalker)
}

func (s *Situation) setStrongLinks(chain *Chain) {
	for _, r := range loop9 {
		for _, c := range loop9 {
			if mask := s.numExcludeMask[r][c]; countTrueBits(mask) == 7 {
				area := AreaOfCell(r, c)
				p1 := FindFirstZero(mask)
				p2 := FindFirstZero(mask | (1 << p1))
				chain.SetStrongLinkAndNeighbors(area, p1, p2)
			}
		}
	}
	for _, n := range loop9 {
		for _, r := range loop9 {
			if mask := s.rowExcludeMask[n][r]; countTrueBits(mask) == 7 {
				area := AreaOfRow(n, r)
				p1 := FindFirstZero(mask)
				p2 := FindFirstZero(mask | (1 << p1))
				chain.SetStrongLinkAndNeighbors(area, p1, p2)
			}
		}
		for _, c := range loop9 {
			if mask := s.colExcludeMask[n][c]; countTrueBits(mask) == 7 {
				area := AreaOfCol(n, c)
				p1 := FindFirstZero(mask)
				p2 := FindFirstZero(mask | (1 << p1))
				chain.SetStrongLinkAndNeighbors(area, p1, p2)
			}
		}
		for _, b := range loop9 {
			if mask := s.blockExcludeMask[n][b]; countTrueBits(mask) == 7 {
				area := AreaOfBlock(n, b)
				p1 := FindFirstZero(mask)
				p2 := FindFirstZero(mask | (1 << p1))
				chain.SetStrongLinkAndNeighbors(area, p1, p2)
			}
		}
	}
}

func (s *Situation) applyWeakLinkExclude(t *Trigger, cw *ChainWalker) (changed int) {
	for _, link := range cw.foundInLoopWeakLinks {
		switch link.Area.AreaDim {
		case DimCell:
			r, c := link.Area.Dim1, link.Area.Dim2
			mask := s.numExcludeMask[r][c]
			if int(countTrueBits(mask))+len(link.Nodes) >= 9 {
				continue
			}
			for n := range link.Nodes {
				mask |= 1 << n
			}
			for {
				n := FindFirstZero(mask)
				if n == -1 {
					break
				}
				mask |= 1 << n
				changed += s.excludeOne(t, RCN(r, c, n))
			}
		case DimRow:
			n, r := link.Area.Dim1, link.Area.Dim2
			mask := s.rowExcludeMask[n][r]
			if int(countTrueBits(mask))+len(link.Nodes) >= 9 {
				continue
			}
			for c := range link.Nodes {
				mask |= 1 << c
			}
			for {
				c := FindFirstZero(mask)
				if c == -1 {
					break
				}
				mask |= 1 << c
				changed += s.excludeOne(t, RCN(r, c, n))
			}
		case DimCol:
			n, c := link.Area.Dim1, link.Area.Dim2
			mask := s.colExcludeMask[n][c]
			if int(countTrueBits(mask))+len(link.Nodes) >= 9 {
				continue
			}
			for r := range link.Nodes {
				mask |= 1 << r
			}
			for {
				r := FindFirstZero(mask)
				if r == -1 {
					break
				}
				mask |= 1 << r
				changed += s.excludeOne(t, RCN(r, c, n))
			}
		case DimBlock:
			n, b := link.Area.Dim1, link.Area.Dim2
			mask := s.blockExcludeMask[n][b]
			if int(countTrueBits(mask))+len(link.Nodes) >= 9 {
				continue
			}
			for p := range link.Nodes {
				mask |= 1 << p
			}
			for {
				p := FindFirstZero(mask)
				if p == -1 {
					break
				}
				mask |= 1 << p
				changed += s.excludeOne(t, BPNtoRCN(b, p, n))
			}
		}
	}
	for _, conflict := range cw.foundConflict {
		if conflict.Val == 0 {
			changed += s.Set(t, conflict.RowColNum)
		} else {
			changed += s.excludeOne(t, conflict.RowColNum)
		}
	}
	return
}

type Node struct {
	RowColNum
	StrongLinks      [4]*Link
	WeakLinks        [4]*Link
	countStrongLinks int
	countWeakLinks   int
	Visited          [2]int
}

type Link struct {
	Area
	Nodes      [9]*Node
	countNodes int
}

func (n *Node) SetStrongLink(link *Link) {
	dim := link.Area.AreaDim
	if n.StrongLinks[dim] == nil {
		n.countStrongLinks++
	}
	n.StrongLinks[dim] = link

	pos := IndexOfDim(link.Area.AreaDim, n.RowColNum)
	if link.Nodes[pos] == nil {
		link.countNodes++
	}
	link.Nodes[pos] = n
}

func (n *Node) SetWeakLink(link *Link) {
	dim := link.Area.AreaDim
	if n.WeakLinks[dim] == nil {
		n.countWeakLinks++
	}
	n.WeakLinks[dim] = link

	pos := IndexOfDim(link.Area.AreaDim, n.RowColNum)
	if link.Nodes[pos] == nil {
		link.countNodes++
	}
	link.Nodes[pos] = n
}

func (n *Node) RemoveStrongLink(link *Link) {
	dim := link.Area.AreaDim
	if n.StrongLinks[dim] != link {
		return
	}
	n.StrongLinks[dim] = nil
	n.countStrongLinks--

	pos := IndexOfDim(link.Area.AreaDim, n.RowColNum)
	if link.Nodes[pos] == n {
		link.Nodes[pos] = nil
		link.countNodes--
	}
}

func (n *Node) RemoveWeakLink(link *Link) {
	dim := link.Area.AreaDim
	if n.WeakLinks[dim] != link {
		return
	}
	n.WeakLinks[dim] = nil
	n.countWeakLinks--

	pos := IndexOfDim(link.Area.AreaDim, n.RowColNum)
	if link.Nodes[pos] == n {
		link.Nodes[pos] = nil
		link.countNodes--
	}
}

type Chain struct {
	Nodes       [9 * 9 * 9]*Node
	StrongLinks [4 * 9 * 9]*Link
	WeakLinks   [4 * 9 * 9]*Link
}

func (c *Chain) getNode(rcn RowColNum) *Node {
	index := rcn.Index()
	node := c.Nodes[index]
	if node == nil {
		node = NewNode()
		node.RowColNum = rcn
		c.Nodes[index] = node
	}
	return node
}

func (c *Chain) getStrongLink(area Area) *Link {
	index := area.Index()
	link := c.StrongLinks[index]
	if link == nil {
		link = NewLink()
		link.Area = area
		c.StrongLinks[index] = link
	}
	return link
}

func (c *Chain) getWeakLink(area Area) *Link {
	index := area.Index()
	link := c.WeakLinks[index]
	if link == nil {
		link = &Link{
			Area: area,
		}
		c.WeakLinks[index] = link
	}
	return link
}

// 创建一个强链、两端节点、节点上的弱链
func (c *Chain) SetStrongLinkAndNeighbors(area Area, p1, p2 int8) {
	// fmt.Printf("SetStrongLinkAndNeighbors(%d,%d,%d,%d,%d)\n", area.AreaDim, area.Dim1+1, area.Dim2+1, p1+1, p2+1)
	strongLink := c.getStrongLink(area)
	node1 := c.SetNodeAndWeakLink(RCNOfAreaPoint(AreaPoint{area, p1}), area.AreaDim)
	node2 := c.SetNodeAndWeakLink(RCNOfAreaPoint(AreaPoint{area, p2}), area.AreaDim)
	node1.SetStrongLink(strongLink)
	node2.SetStrongLink(strongLink)
}

// 创建一个节点，和节点上的弱链
func (c *Chain) SetNodeAndWeakLink(rcn RowColNum, exceptDim Dim) *Node {
	// fmt.Printf("SetNodeAndWeakLink(%d,%d,%d,%d)\n", rcn.Row+1, rcn.Col+1, rcn.Num+1, exceptDim)
	node := c.getNode(rcn)
	for _, d := range dims {
		if d == exceptDim {
			continue
		}
		ap := AreaPointOfRCN(rcn, d)
		weakLink := c.getWeakLink(ap.Area)
		node.SetWeakLink(weakLink)
	}
	return node
}

// 递归删除所有度为1的节点、强链、弱链（为了找到所有在环内的节点）
func (c *Chain) Prune() {
	for _, link := range c.WeakLinks {
		c.pruneWeakLink(link)
	}
}

func (c *Chain) pruneStrongLink(link *Link) {
	if link.countNodes > 1 {
		return
	}
	index := link.Area.Index()
	if c.StrongLinks[index] != link {
		return
	}
	// fmt.Printf("pruneStrongLink(%d,%d,%d)\n", link.Area.AreaDim, link.Area.Dim1+1, link.Area.Dim2+1)
	c.StrongLinks[index] = nil
	for _, node := range link.Nodes {
		node.RemoveStrongLink(link)
		c.pruneNode(node)
	}
	ReleaseLink(link)
}

func (c *Chain) pruneWeakLink(link *Link) {
	if len(link.Nodes) > 1 {
		return
	}
	index := link.Area.Index()
	if c.WeakLinks[index] != link {
		return
	}
	// fmt.Printf("pruneWeakLink(%d,%d,%d)\n", link.Area.AreaDim, link.Area.Dim1+1, link.Area.Dim2+1)
	c.WeakLinks[index] = nil
	for _, node := range link.Nodes {
		node.RemoveWeakLink(link)
		c.pruneNode(node)
	}
	ReleaseLink(link)
}

func (c *Chain) pruneNode(node *Node) {
	if node.countStrongLinks > 0 || node.countWeakLinks > 0 {
		return
	}
	index := node.RowColNum.Index()
	if c.Nodes[index] != node {
		return
	}
	// fmt.Printf("pruneNode(%d,%d,%d)\n", node.Row+1, node.Col+1, node.Num+1)
	c.Nodes[index] = nil
	for _, link := range node.StrongLinks {
		node.RemoveStrongLink(link)
		c.pruneStrongLink(link)
	}
	for _, link := range node.WeakLinks {
		node.RemoveWeakLink(link)
		c.pruneWeakLink(link)
	}
	ReleaseNode(node)
}

type ChainWalker struct {
	*Chain
	foundInLoopWeakLinks []*Link
	foundConflict        []RowColNumVal
}

func (cw *ChainWalker) WalkAll() {
	for _, node := range cw.Nodes {
		if node == nil {
			continue
		}
		cw.Walk(node, 1, 0)
		cw.Walk(node, 1, 1)
	}
}

// 递归标记所有节点，标记环内的链
// 返回-1表示没有环，正数表示访问过且在当前路径上（环的一部分）
func (cw *ChainWalker) Walk(node *Node, deep int, layer int8) (foundParent int, foundConflict int) {
	switch {
	case node.Visited[layer] < 0:
		return -1, -1
	case node.Visited[layer] > 0:
		return node.Visited[layer], -1
	}
	foundConflict = -1
	if node.Visited[1-layer] > 0 {
		foundConflict = node.Visited[1-layer]
	}
	node.Visited[layer] = deep
	foundParent = deep + 1
	var walkingLinks [4]*Link
	switch layer {
	case 0:
		walkingLinks = node.StrongLinks
	case 1:
		walkingLinks = node.WeakLinks
	}
	for _, link := range walkingLinks {
		if link == nil {
			continue
		}
		for _, nextNode := range link.Nodes {
			if nextNode == node || nextNode == nil {
				continue
			}
			parent, conflict2 := cw.Walk(nextNode, deep+1, 1-layer)
			if conflict2 > foundConflict {
				foundConflict = conflict2
			}
			if parent > 0 && parent <= deep {
				if parent < foundParent {
					foundParent = parent
				}
				if layer == 1 {
					cw.foundInLoopWeakLinks = append(cw.foundInLoopWeakLinks, link)
				}
			}
		}
	}
	node.Visited[layer] = -1
	if foundParent >= deep {
		foundParent = -1
	}
	if foundConflict >= deep {
		cw.foundConflict = append(cw.foundConflict, RowColNumVal{node.RowColNum, layer})
	}
	return
}
