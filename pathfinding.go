package main

func canOccupy(centerX, centerY float64, width, height float64) bool {
	left, top := centerX-width/2, centerY-height/2
	right, bottom := centerX+width/2, centerY+height/2
	if left < 32 || top < 48 || right > 928 || bottom > 496 {
		return false
	}
	for _, wall := range ruinWalls {
		wallLeft, wallTop := wall[0]-2, wall[1]-2
		wallRight, wallBottom := wall[0]+wall[2]+2, wall[1]+wall[3]+2
		if left < wallRight && right > wallLeft && top < wallBottom && bottom > wallTop {
			return false
		}
	}
	return true
}

func worldToCell(x, y float64) (Cell, bool) {
	cell := Cell{x: int((x - 32) / tileSize), y: int((y - 48) / tileSize)}
	return cell, cell.x >= 0 && cell.x < gridWidth && cell.y >= 0 && cell.y < gridHeight
}

func worldToCellOrDefault(x, y float64) Cell {
	cell, ok := worldToCell(x, y)
	if !ok {
		return Cell{x: gridWidth / 2, y: gridHeight / 2}
	}
	return cell
}

func cellCenter(cell Cell) (float64, float64) {
	return float64(32 + cell.x*tileSize + tileSize/2), float64(48 + cell.y*tileSize + tileSize/2)
}

func isBlocked(cell Cell) bool {
	if cell.x < 0 || cell.x >= gridWidth || cell.y < 0 || cell.y >= gridHeight {
		return true
	}
	cellLeft := float64(32 + cell.x*tileSize)
	cellTop := float64(48 + cell.y*tileSize)
	cellRight := cellLeft + tileSize
	cellBottom := cellTop + tileSize
	for _, wall := range ruinWalls {
		wallLeft, wallTop := wall[0]-2, wall[1]-2
		wallRight, wallBottom := wall[0]+wall[2]+2, wall[1]+wall[3]+2
		if cellLeft < wallRight && cellRight > wallLeft && cellTop < wallBottom && cellBottom > wallTop {
			return true
		}
	}
	return false
}

func nearestWalkableCell(target Cell) Cell {
	best := target
	bestDistance := 9999
	for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			candidate := Cell{x: x, y: y}
			if isBlocked(candidate) {
				continue
			}
			distance := absInt(candidate.x-target.x) + absInt(candidate.y-target.y)
			if distance < bestDistance {
				best, bestDistance = candidate, distance
			}
		}
	}
	return best
}

func findPath(start, target Cell) []Cell {
	if isBlocked(start) || isBlocked(target) {
		return nil
	}
	visited := [gridHeight][gridWidth]bool{}
	previous := [gridHeight][gridWidth]Cell{}
	queue := []Cell{start}
	visited[start.y][start.x] = true
	previous[start.y][start.x] = start
	directions := [...]Cell{{x: 1}, {x: -1}, {y: 1}, {y: -1}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == target {
			break
		}
		for _, direction := range directions {
			next := Cell{x: current.x + direction.x, y: current.y + direction.y}
			if isBlocked(next) || visited[next.y][next.x] {
				continue
			}
			visited[next.y][next.x] = true
			previous[next.y][next.x] = current
			queue = append(queue, next)
		}
	}
	if !visited[target.y][target.x] {
		return nil
	}
	path := []Cell{}
	for current := target; ; current = previous[current.y][current.x] {
		path = append(path, current)
		if current == start {
			break
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
