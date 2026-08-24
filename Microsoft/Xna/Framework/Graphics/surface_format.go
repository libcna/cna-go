package graphics

// SurfaceFormat identifies the format of surface data.
type SurfaceFormat int32

const (
	SurfaceFormatColor           SurfaceFormat = 0
	SurfaceFormatBgr565          SurfaceFormat = 1
	SurfaceFormatBgra5551        SurfaceFormat = 2
	SurfaceFormatBgra4444        SurfaceFormat = 3
	SurfaceFormatDxt1            SurfaceFormat = 4
	SurfaceFormatDxt3            SurfaceFormat = 5
	SurfaceFormatDxt5            SurfaceFormat = 6
	SurfaceFormatNormalizedByte2 SurfaceFormat = 7
	SurfaceFormatNormalizedByte4 SurfaceFormat = 8
	SurfaceFormatRgba1010102     SurfaceFormat = 9
	SurfaceFormatRg32            SurfaceFormat = 10
	SurfaceFormatRgba64          SurfaceFormat = 11
	SurfaceFormatAlpha8          SurfaceFormat = 12
	SurfaceFormatSingle          SurfaceFormat = 13
	SurfaceFormatVector2         SurfaceFormat = 14
	SurfaceFormatVector4         SurfaceFormat = 15
	SurfaceFormatHalfSingle      SurfaceFormat = 16
	SurfaceFormatHalfVector2     SurfaceFormat = 17
	SurfaceFormatHalfVector4     SurfaceFormat = 18
	SurfaceFormatHdrBlendable    SurfaceFormat = 19
)
