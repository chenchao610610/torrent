package torrent

import (
	"strings"

	"github.com/anacrolix/torrent/metainfo"
)

// Provides access to regions of torrent data that correspond to its files.
type File struct {
	t      *Torrent
	path   string
	offset int64
	length int64
	fi     metainfo.FileInfo
}

func (f *File) Torrent() *Torrent {
	return f.t
}

// Data for this file begins this many bytes into the Torrent.
func (f *File) Offset() int64 {
	return f.offset
}

// The FileInfo from the metainfo.Info to which this file corresponds.
func (f File) FileInfo() metainfo.FileInfo {
	return f.fi
}

// The file's path components joined by '/'.
func (f File) Path() string {
	return f.path
}

// The file's length in bytes.
func (f *File) Length() int64 {
	return f.length
}

// The relative file path for a multi-file torrent, and the torrent name for a
// single-file torrent.
func (f *File) DisplayPath() string {
	fip := f.FileInfo().Path
	if len(fip) == 0 {
		return f.t.Info().Name
	}
	return strings.Join(fip, "/")

}

// The download status of a piece that comprises part of a File.
type FilePieceState struct {
	Bytes int64 // Bytes within the piece that are part of this File.
	PieceState
}

// Returns the state of pieces in this file.
func (f *File) State() (ret []FilePieceState) {
	f.t.cl.mu.RLock()
	defer f.t.cl.mu.RUnlock()
	pieceSize := int64(f.t.usualPieceSize())
	off := f.offset % pieceSize
	remaining := f.length
	for i := int(f.offset / pieceSize); ; i++ {
		if remaining == 0 {
			break
		}
		len1 := pieceSize - off
		if len1 > remaining {
			len1 = remaining
		}
		ps := f.t.pieceState(i)
		ret = append(ret, FilePieceState{len1, ps})
		off = 0
		remaining -= len1
	}
	return
}

// Requests that all pieces containing data in the file be downloaded.
func (f *File) Download() {
	f.t.DownloadPieces(f.t.byteRegionPieces(f.offset, f.length))
}

// Deprecated: Use File.DownloadRegion.
func (f *File) PrioritizeRegion(off, len int64) {
	f.DownloadRegion(off, len)
}

// Requests that torrent pieces containing bytes in the given region of the
// file be downloaded.
func (f *File) DownloadRegion(off, len int64) {
	f.t.DownloadPieces(f.t.byteRegionPieces(f.offset+off, len))
}

func byteRegionExclusivePieces(off, size, pieceSize int64) (begin, end int) {
	begin = int((off + pieceSize - 1) / pieceSize)
	end = int((off + size) / pieceSize)
	return
}

func (f *File) exclusivePieces() (begin, end int) {
	return byteRegionExclusivePieces(f.offset, f.length, int64(f.t.usualPieceSize()))
}

func (f *File) Cancel() {
	f.t.CancelPieces(f.exclusivePieces())
}

func (f *File) NewReader() Reader {
	tr := reader{
		mu:        &f.t.cl.mu,
		t:         f.t,
		readahead: 5 * 1024 * 1024,
		offset:    f.Offset(),
		length:    f.Length(),
	}
	f.t.addReader(&tr)
	return &tr
}
