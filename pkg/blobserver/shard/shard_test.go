/*
Copyright 2016 The Camlistore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shard

import (
	"testing"

	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/blobserver/storagetest"
	"camlistore.org/pkg/test"
)

type testStorage struct {
	sto     *shardStorage
	shards  []*test.Fetcher
	t       *testing.T
	limited map[blobserver.Storage]uint64
}

func newTestStorage(t *testing.T) *testStorage {
	a, b := &test.Fetcher{}, &test.Fetcher{}
	sto := &shardStorage{
		shardPrefixes: []string{
			"/a/",
			"/b/",
		},
		shards: []blobserver.Storage{a, b},
	}

	sto.selector = RegularBackendSelector(sto.shards)

	ts := &testStorage{
		sto:    sto,
		shards: []*test.Fetcher{a, b},
		t:      t,
		limited: map[blobserver.Storage]uint64{
			a: 1 << 16,
			b: 1 << 15,
		},
	}

	return ts
}

func TestShardBasic(t *testing.T) {
	storagetest.Test(t, func(t *testing.T) (sto blobserver.Storage, cleanup func()) {
		return newTestStorage(t).sto, nil
	})
}

func TestSizeWeightedWithLimit(t *testing.T) {
	storagetest.Test(t, func(t *testing.T) (sto blobserver.Storage, cleanup func()) {
		tstore := newTestStorage(t)
		store := tstore.sto
		store.selector = NewSizeWeightedBackendSelectorWithLimits(tstore.limited)
		return store, nil
	})
}

func TestShardRegular(t *testing.T) {
	thingA := &test.Blob{"something"}
	thingB := &test.Blob{"something else"}

	ts := newTestStorage(t)

	ts.sto.ReceiveBlob(thingB.BlobRef(), thingB.Reader())
	ts.sto.ReceiveBlob(thingA.BlobRef(), thingA.Reader())

	// sha1-1af17e73721dbe0c40011b82ed4bb1a7dbe3ce29
	// sum32: 452034163
	ts.checkShard(thingA, 1)

	// sha1-637828c03aae38af639cc721200f2584864e8797
	// sum32: 1668819136
	ts.checkShard(thingB, 0)
}

func TestWeighted(t *testing.T) {
	ts := newTestStorage(t)
	ts.sto.selector = NewSizeWeightedBackendSelectorWithLimits(ts.limited)

	thingA := &test.Blob{"something"}
	thingB := &test.Blob{"something else"}
	thingC := &test.Blob{"again something else."}

	ts.sto.ReceiveBlob(thingA.BlobRef(), thingA.Reader())
	ts.sto.ReceiveBlob(thingB.BlobRef(), thingB.Reader())
	ts.sto.ReceiveBlob(thingC.BlobRef(), thingC.Reader())

	ts.checkShard(thingA, 0)
	ts.checkShard(thingB, 0)
	ts.checkShard(thingC, 1)
}

// checkShard iterates through shards and find the blob. error if it is not found in expectShard, found somewhere else, or not found at all
func (sto testStorage) checkShard(b *test.Blob, expectShard int) {
	for shardN, shard := range sto.shards {
		_, _, err := shard.Fetch(b.BlobRef())
		if err != nil && shardN == expectShard {
			sto.t.Errorf("expected ref %v in shard %d, but didn't find it there", b.BlobRef(), expectShard)
			continue
		}

		if err != nil {
			// node wasn't found here, as expected
			continue
		}

		if shardN != expectShard {
			sto.t.Errorf("found ref %v in shard %d, expected in shard %d", b.BlobRef(), shardN, expectShard)
			continue
		}

		// node was found, and we expected it
	}
}
