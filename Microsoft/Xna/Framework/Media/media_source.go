package media

import (
	"github.com/openeggbert/cna-go/internal/interop"
)

// MediaSource is Microsoft.Xna.Framework.Media.MediaSource: where a media
// library's contents came from.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It is a VALUE, not a handle, and that is measured
//
// Every other type in this namespace owns a native object and implements
// IDisposable. This one declares neither: its whole contract is a name, a kind,
// a ToString and one static that enumerates the available sources.
//
// CNA agrees. There is no CNA_MediaSourceHandle -- the available sources are
// addressed by INDEX (`cna_media_source_get_type_at`,
// `cna_media_source_copy_name_at`), and an open library answers its own
// source's type and name off the LIBRARY handle. So the projection holds the
// two values rather than a handle, and every member is a field read.
type MediaSource struct {
	sourceType MediaSourceType
	name       string
	// index is the position this source has in the available list, and
	// enumerated says whether it has one at all.
	//
	// A source from MediaSourceGetAvailableMediaSources carries its index,
	// which is what MediaLibrary's constructor needs. One read back from an
	// open library does NOT: CNA answers that library's source by value and
	// never says where it sat in the list.
	index      uint32
	enumerated bool
}

// MediaSourceType is MediaSource::get_MediaSourceType, one field read.
func (m *MediaSource) MediaSourceType() MediaSourceType {
	if m == nil {
		return MediaSourceTypeLocalDevice
	}
	return m.sourceType
}

// Name is MediaSource::get_Name, one field read.
func (m *MediaSource) Name() string {
	if m == nil {
		return ""
	}
	return m.name
}

// ToString is MediaSource::ToString(), which the reference implements as the
// name.
func (m *MediaSource) ToString() string { return m.Name() }

// MediaSourceGetAvailableMediaSources is
// MediaSource::GetAvailableMediaSources(), the one static.
//
// It is a package function because the CLR member is static, which is the
// settled spelling. The result is a LIST in the contract -- IList<MediaSource>
// -- which projects to a Go slice.
//
// A host with no media backend answers an EMPTY list rather than an error, and
// that is the expected outcome rather than a gap: the reference answers an
// empty list on a device with no media sources too.
func MediaSourceGetAvailableMediaSources() ([]*MediaSource, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errMediaNoRuntime
	}
	count, err := runtime.MediaSourceAvailableCount()
	if err != nil {
		return nil, err
	}
	sources := make([]*MediaSource, 0, count)
	for index := uint32(0); index < count; index++ {
		kind, err := runtime.MediaSourceTypeAt(index)
		if err != nil {
			return nil, err
		}
		name, err := runtime.MediaSourceNameAt(index)
		if err != nil {
			return nil, err
		}
		sources = append(sources, &MediaSource{
			sourceType: MediaSourceType(kind),
			name:       name,
			index:      index,
			enumerated: true,
		})
	}
	return sources, nil
}
