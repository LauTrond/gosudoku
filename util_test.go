package main

import (
	"fmt"
	"testing"
)

func TestChain(t *testing.T) {
	var c Chain

	c.SetStrongLinkAndNeighbors(Area{AreaDim: DimCell, Dim1: 0, Dim2: 0}, 0, 2)
	c.SetStrongLinkAndNeighbors(Area{AreaDim: DimCell, Dim1: 0, Dim2: 1}, 0, 2)
	c.SetStrongLinkAndNeighbors(Area{AreaDim: DimBlock, Dim1: 0, Dim2: 0}, 0, 2)
	c.SetStrongLinkAndNeighbors(Area{AreaDim: DimBlock, Dim1: 0, Dim2: 1}, 0, 2)
	c.Prune()
	for _, link := range c.WeakLinks {
		fmt.Printf("WeakLink: %d,%d,%d\n", link.Area.AreaDim, link.Area.Dim1+1, link.Area.Dim2+1)
	}
}
