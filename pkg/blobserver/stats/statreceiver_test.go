package stats

import (
	"testing"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/test"
	"golang.org/x/net/context"
)

func TestStats(t *testing.T) {
	st := &Receiver{}
	foo := test.Blob{"foo"}
	bar := test.Blob{"bar"}
	foobar := test.Blob{"foobar"}

	foo.MustUpload(t, st)
	bar.MustUpload(t, st)
	foobar.MustUpload(t, st)

	if st.NumBlobs() != 3 {
		t.Fatal("expected 3 stats")
	}

	sizes := st.Sizes()
	if sizes[0] != 3 || sizes[1] != 3 || sizes[2] != 6 {
		t.Fatal("stats reported the incorrect sizes:", sizes)
	}

	if st.SumBlobSize() != 12 {
		t.Fatal("stats reported the incorrect sum sizes:", st.SumBlobSize())
	}

	dest := make(chan blob.SizedRef, 5) // buffer past what we expect so we see if there is something extra
	err := st.StatBlobs(dest, []blob.Ref{
		foo.BlobRef(),
		bar.BlobRef(),
		foobar.BlobRef(),
	})

	if err != nil {
		t.Fatal(err)
	}

	var foundFoo, foundBar, foundFoobar bool

	func() {
		for {
			select {
			case sb := <-dest:
				switch {
				case sb.Ref == foo.BlobRef():
					foundFoo = true
				case sb.Ref == bar.BlobRef():
					foundBar = true
				case sb.Ref == foobar.BlobRef():
					foundFoobar = true
				default:
					t.Fatal("found unexpected ref:", sb)
				}
			default:
				return
			}
		}
	}()

	if !foundFoo || !foundBar || !foundFoobar {
		t.Fatalf("missing a ref: foo: %t bar: %t foobar: %t", foundFoo, foundBar, foundFoobar)
	}

	dest = make(chan blob.SizedRef, 2)
	err = st.EnumerateBlobs(context.Background(), dest, "sha1-7", 2)
	if err != nil {
		t.Fatal(err)
	}

	expectFoobar := <-dest
	if expectFoobar.Ref != foobar.BlobRef() {
		t.Fatal("expected foobar")
	}

	val, expectFalse := <-dest
	if expectFalse != false {
		t.Fatal("expected dest to be closed, saw", val)
	}

	err = st.RemoveBlobs([]blob.Ref{
		foo.BlobRef(),
		bar.BlobRef(),
		foobar.BlobRef(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if st.NumBlobs() != 0 {
		t.Fatal("all blobs should be gone, instead we have", st.NumBlobs())
	}
}
