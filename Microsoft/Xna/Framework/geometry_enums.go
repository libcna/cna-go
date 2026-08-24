package framework

// ContainmentType classifies one volume relative to another.
type ContainmentType int32

const (
	ContainmentTypeDisjoint   ContainmentType = 0
	ContainmentTypeContains   ContainmentType = 1
	ContainmentTypeIntersects ContainmentType = 2
)

// PlaneIntersectionType classifies geometry against a plane.
type PlaneIntersectionType int32

const (
	PlaneIntersectionTypeFront        PlaneIntersectionType = 0
	PlaneIntersectionTypeBack         PlaneIntersectionType = 1
	PlaneIntersectionTypeIntersecting PlaneIntersectionType = 2
)
