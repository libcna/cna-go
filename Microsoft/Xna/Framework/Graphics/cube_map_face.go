package graphics

// CubeMapFace identifies one face of an XNA cube map.
type CubeMapFace int32

const (
	CubeMapFacePositiveX CubeMapFace = 0
	CubeMapFaceNegativeX CubeMapFace = 1
	CubeMapFacePositiveY CubeMapFace = 2
	CubeMapFaceNegativeY CubeMapFace = 3
	CubeMapFacePositiveZ CubeMapFace = 4
	CubeMapFaceNegativeZ CubeMapFace = 5
)
