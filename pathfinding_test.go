package main

import "testing"

func TestWorldCellRoundTripUsesNavigationGrid(t *testing.T) {
	cell, ok := worldToCell(40, 56)
	if !ok {
		t.Fatal("expected point inside the world bounds")
	}
	if cell != (Cell{x: 0, y: 0}) {
		t.Fatalf("world point mapped to %+v, want cell (0,0)", cell)
	}
	x, y := cellCenter(cell)
	if x != 40 || y != 56 {
		t.Fatalf("cell center was (%v,%v), want (40,56)", x, y)
	}
}

func TestFindPathUsesFineGrid(t *testing.T) {
	path := findPath(Cell{x: 0, y: 0}, Cell{x: 2, y: 1})
	if len(path) != 4 {
		t.Fatalf("path has %d cells, want 4", len(path))
	}
	if path[0] != (Cell{x: 0, y: 0}) || path[len(path)-1] != (Cell{x: 2, y: 1}) {
		t.Fatalf("path endpoints were %+v -> %+v", path[0], path[len(path)-1])
	}
}
