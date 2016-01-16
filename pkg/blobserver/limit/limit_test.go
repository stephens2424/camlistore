package limit

import (
	"testing"

	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/blobserver/memory"
	"camlistore.org/pkg/blobserver/storagetest"
	"camlistore.org/pkg/test"
)

// TestLimitStorage tests against an in-memory blobserver.
func TestLimitStorageBasic(t *testing.T) {
	storagetest.Test(t, func(t *testing.T) (blobserver.Storage, func()) {
		return &Storage{maxSize: 1 << 4, Storage: &memory.Storage{}}, func() {}
	})
}

func TestLimitStorage(t *testing.T) {
	s := &Storage{maxSize: 10, Storage: &memory.Storage{}}

	a := test.Blob{"a"}
	b := test.Blob{"big blob is too big for the limited storage"}

	sr, err := s.ReceiveBlob(a.BlobRef(), a.Reader())
	if err != nil {
		t.Error(err)
	}
	if sr.Size != 1 {
		t.Error("received too many bytes:", sr.Size)
	}

	sr, err = s.ReceiveBlob(b.BlobRef(), b.Reader())
	switch err {
	case nil:
		t.Error("blob b should have been too big")
	default:
		t.Error(err)
	case ErrTooBig:
		// pass
	}
}
