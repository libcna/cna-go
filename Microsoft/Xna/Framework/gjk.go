package framework

var gjkBitsToIndices = [16]int{0, 1, 2, 17, 3, 25, 26, 209, 4, 33, 34, 273, 35, 281, 282, 2257}

type gjk struct {
	closest           Vector3
	y                 [4]Vector3
	yLengthSquared    [4]float32
	edges             [4][4]Vector3
	edgeLengthSquared [4][4]float32
	determinants      [16][4]float32
	simplexBits       int
	maxLengthSquared  float32
}

func (g *gjk) reset()             { g.simplexBits = 0; g.maxLengthSquared = 0 }
func (g *gjk) fullSimplex() bool  { return g.simplexBits == 15 }
func gjkDot(a, b Vector3) float32 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (g *gjk) addSupportPoint(point Vector3) bool {
	newIndex := (gjkBitsToIndices[g.simplexBits^15] & 7) - 1
	g.y[newIndex] = point
	g.yLengthSquared[newIndex] = point.LengthSquared()
	for indices := gjkBitsToIndices[g.simplexBits]; indices != 0; indices >>= 3 {
		index := (indices & 7) - 1
		edge := Vector3SubtractByVector3AndVector3(g.y[index], point)
		g.edges[index][newIndex] = edge
		g.edges[newIndex][index] = Vector3NegateByVector3(edge)
		g.edgeLengthSquared[newIndex][index] = edge.LengthSquared()
		g.edgeLengthSquared[index][newIndex] = g.edgeLengthSquared[newIndex][index]
	}
	g.updateDeterminant(newIndex)
	return g.updateSimplex(newIndex)
}
func (g *gjk) updateDeterminant(newIndex int) {
	newBit := 1 << newIndex
	g.determinants[newBit][newIndex] = 1
	all := gjkBitsToIndices[g.simplexBits]
	remaining := all
	prior := 0
	for remaining != 0 {
		index := (remaining & 7) - 1
		indexBit := 1 << index
		pair := indexBit | newBit
		g.determinants[pair][index] = gjkDot(g.edges[newIndex][index], g.y[newIndex])
		g.determinants[pair][newIndex] = gjkDot(g.edges[index][newIndex], g.y[index])
		earlier := all
		for j := 0; j < prior; j++ {
			ei := (earlier & 7) - 1
			eb := 1 << ei
			triple := pair | eb
			edgeIndex := newIndex
			if g.edgeLengthSquared[index][ei] < g.edgeLengthSquared[newIndex][ei] {
				edgeIndex = index
			}
			g.determinants[triple][ei] = g.determinants[pair][index]*gjkDot(g.edges[edgeIndex][ei], g.y[index]) + g.determinants[pair][newIndex]*gjkDot(g.edges[edgeIndex][ei], g.y[newIndex])
			edgeIndex = newIndex
			if g.edgeLengthSquared[ei][index] < g.edgeLengthSquared[newIndex][index] {
				edgeIndex = ei
			}
			g.determinants[triple][index] = g.determinants[eb|newBit][ei]*gjkDot(g.edges[edgeIndex][index], g.y[ei]) + g.determinants[eb|newBit][newIndex]*gjkDot(g.edges[edgeIndex][index], g.y[newIndex])
			edgeIndex = ei
			if g.edgeLengthSquared[index][newIndex] < g.edgeLengthSquared[ei][newIndex] {
				edgeIndex = index
			}
			g.determinants[triple][newIndex] = g.determinants[indexBit|eb][ei]*gjkDot(g.edges[edgeIndex][newIndex], g.y[ei]) + g.determinants[indexBit|eb][index]*gjkDot(g.edges[edgeIndex][newIndex], g.y[index])
			earlier >>= 3
		}
		remaining >>= 3
		prior++
	}
	if g.simplexBits|newBit != 15 {
		return
	}
	selected := 3
	if !(g.edgeLengthSquared[1][0] < g.edgeLengthSquared[2][0]) {
		if g.edgeLengthSquared[2][0] < g.edgeLengthSquared[3][0] {
			selected = 2
		}
	} else if g.edgeLengthSquared[1][0] < g.edgeLengthSquared[3][0] {
		selected = 1
	}
	g.determinants[15][0] = g.determinants[14][1]*gjkDot(g.edges[selected][0], g.y[1]) + g.determinants[14][2]*gjkDot(g.edges[selected][0], g.y[2]) + g.determinants[14][3]*gjkDot(g.edges[selected][0], g.y[3])
	selected = 0
	if !(g.edgeLengthSquared[0][1] < g.edgeLengthSquared[2][1]) {
		if g.edgeLengthSquared[2][1] < g.edgeLengthSquared[3][1] {
			selected = 2
		} else {
			selected = 3
		}
	} else if !(g.edgeLengthSquared[0][1] < g.edgeLengthSquared[3][1]) {
		selected = 3
	}
	g.determinants[15][1] = g.determinants[13][0]*gjkDot(g.edges[selected][1], g.y[0]) + g.determinants[13][2]*gjkDot(g.edges[selected][1], g.y[2]) + g.determinants[13][3]*gjkDot(g.edges[selected][1], g.y[3])
	selected = 0
	if !(g.edgeLengthSquared[0][2] < g.edgeLengthSquared[1][2]) {
		if g.edgeLengthSquared[1][2] < g.edgeLengthSquared[3][2] {
			selected = 1
		} else {
			selected = 3
		}
	} else if !(g.edgeLengthSquared[0][2] < g.edgeLengthSquared[3][2]) {
		selected = 3
	}
	g.determinants[15][2] = g.determinants[11][0]*gjkDot(g.edges[selected][2], g.y[0]) + g.determinants[11][1]*gjkDot(g.edges[selected][2], g.y[1]) + g.determinants[11][3]*gjkDot(g.edges[selected][2], g.y[3])
	selected = 0
	if !(g.edgeLengthSquared[0][3] < g.edgeLengthSquared[1][3]) {
		if g.edgeLengthSquared[1][3] < g.edgeLengthSquared[2][3] {
			selected = 1
		} else {
			selected = 2
		}
	} else if !(g.edgeLengthSquared[0][3] < g.edgeLengthSquared[2][3]) {
		selected = 2
	}
	g.determinants[15][3] = g.determinants[7][0]*gjkDot(g.edges[selected][3], g.y[0]) + g.determinants[7][1]*gjkDot(g.edges[selected][3], g.y[1]) + g.determinants[7][2]*gjkDot(g.edges[selected][3], g.y[2])
}
func (g *gjk) satisfies(candidate, all int) bool {
	for indices := gjkBitsToIndices[all]; indices != 0; indices >>= 3 {
		index := (indices & 7) - 1
		bit := 1 << index
		if bit&candidate != 0 {
			if g.determinants[candidate][index] <= 0 {
				return false
			}
		} else if g.determinants[candidate|bit][index] > 0 {
			return false
		}
	}
	return true
}
func (g *gjk) updateSimplex(newIndex int) bool {
	all := g.simplexBits | (1 << newIndex)
	newBit := 1 << newIndex
	for bits := g.simplexBits; bits != 0; bits-- {
		if bits&all == bits && g.satisfies(bits|newBit, all) {
			g.simplexBits = bits | newBit
			g.closest = g.computeClosestPoint()
			return true
		}
	}
	if !g.satisfies(newBit, all) {
		return false
	}
	g.simplexBits = newBit
	g.closest = g.y[newIndex]
	g.maxLengthSquared = g.yLengthSquared[newIndex]
	return true
}
func (g *gjk) computeClosestPoint() Vector3 {
	sum := float32(0)
	r := Vector3{}
	g.maxLengthSquared = 0
	for indices := gjkBitsToIndices[g.simplexBits]; indices != 0; indices >>= 3 {
		index := (indices & 7) - 1
		d := g.determinants[g.simplexBits][index]
		sum += d
		r = Vector3AddByVector3AndVector3(r, Vector3MultiplyByVector3AndSingle(g.y[index], d))
		g.maxLengthSquared = MathHelperMax(g.maxLengthSquared, g.yLengthSquared[index])
	}
	return Vector3DivideByVector3AndSingle(r, sum)
}
