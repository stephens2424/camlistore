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
	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/blobserver/limit"
)

type BackendSelector interface {
	SelectBackend(b blob.Ref) uint32
}

type BackendSelectorFunc func(b blob.Ref) uint32

func (f BackendSelectorFunc) SelectBackend(b blob.Ref) uint32 {
	return f(b)
}

func RegularBackendSelector(bs []blobserver.Storage) BackendSelector {
	div := uint32(len(bs))
	return BackendSelectorFunc(func(b blob.Ref) uint32 {
		return b.Sum32() % div
	})
}

type SizeWeightedBackendSelector []*limit.Storage

func NewSizeWeightedBackendSelector(bs []*limit.Storage) BackendSelector {
	return SizeWeightedBackendSelector(bs)
}

func NewSizeWeightedBackendSelectorWithLimits(bs map[blobserver.Storage]uint64) BackendSelector {
	limits := []*limit.Storage{}
	for sto, lim := range bs {
		sto := limit.NewLimit(lim, sto)
		limits = append(limits, sto)
	}
	return SizeWeightedBackendSelector(limits)
}

const MaxUint64 = ^uint64(0)

// SelectBackend chooses a shard for b predictably, spreading blobs evenly across a hashspace
// and proportionally, by size, among s's backends.
func (s SizeWeightedBackendSelector) SelectBackend(b blob.Ref) uint32 {
	const hashspace = MaxUint64

	totalCap := float64(s.totalCapacity())

	sum := b.Sum64()
	sumPct := float64(sum) / float64(hashspace)

	var end uint64
	for i, st := range s {
		// update the cumulative ending capacity including this new shard
		end += st.Capacity()

		// get the cumulative percentage of total capacity
		capPct := float64(end) / totalCap

		// if the sum percentage is less than the cumulative capacity percentage,
		// we have found the shard that corresponds to this position in the hashspace.
		if sumPct <= capPct {
			return uint32(i)
		}
	}

	panic("union: shard not selected for size weighted union")
}

func (s SizeWeightedBackendSelector) totalCapacity() uint64 {
	var total uint64
	for _, st := range s {
		total += st.Capacity()
	}
	return total
}
