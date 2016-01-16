package limit

import (
	"errors"
	"io"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
	"go4.org/jsonconfig"
)

// Storage implements blobserver.Storage with a byte limit
type Storage struct {
	maxSize, size uint64 // zero really means zero bytes

	blobserver.Storage
}

func NewLimit(max uint64, sto blobserver.Storage) *Storage {
	return &Storage{
		maxSize: max,
		Storage: sto,
	}
}

// Consumed implements blobserver.StorageStatter
func (s *Storage) Consumed() uint64 {
	return s.size
}

func (s *Storage) Capacity() uint64 {
	return s.maxSize
}

func (s *Storage) Available() uint64 {
	return s.maxSize - s.size
}

func init() {
	blobserver.RegisterStorageConstructor("limit", blobserver.StorageConstructor(newFromConfig))
}

func newFromConfig(_ blobserver.Loader, config jsonconfig.Obj) (blobserver.Storage, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Storage{}, nil
}

var ErrTooBig = errors.New("blobserver/limit: blob exceeds blobserver limit")

// ReceiveBlob implements blobserver.BlobReceiver
func (s *Storage) ReceiveBlob(br blob.Ref, source io.Reader) (blob.SizedRef, error) {
	// the limit is our max size, plus the current capacity
	limit := s.maxSize - s.size

	// only read up to the limit
	sr, err := s.Storage.ReceiveBlob(br, io.LimitReader(source, int64(limit)))

	// if sr.Size is 0 or is *at* the read limit, try to read one more byte
	// from the original source. If there's another byte available, we hit
	// the limit and the blob is too big
	if uint64(sr.Size) == limit || sr.Size == 0 {
		n, err := source.Read(make([]byte, 1))

		// if we read any bytes or there was an error other than EOF, the blob is
		// too big
		if n > 0 || err != io.EOF {
			return sr, ErrTooBig
		}
	}

	if err != nil {
		return sr, err
	}

	return sr, nil
}
