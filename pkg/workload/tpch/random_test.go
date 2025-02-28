// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package tpch

import (
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/bufalloc"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

func TestRandPartName(t *testing.T) {
	var a bufalloc.ByteAllocator
	rng := rand.New(rand.NewSource(uint64(timeutil.Now().UnixNano())))
	seen := make(map[string]int)
	runOneRound := func() {
		namePerm := make([]int, len(randPartNames))
		for i := range namePerm {
			namePerm[i] = i
		}
		res := randPartName(rng, namePerm, &a)
		names := strings.Split(string(res), " ")
		assert.Equal(t, len(names), nPartNames)
		seenLocal := make(map[string]int)
		for _, name := range names {
			if _, ok := seenLocal[name]; ok {
				t.Errorf("names in '%s' are not unique", res)
			}
			seenLocal[name]++
			seen[name]++
		}
	}

	// We can't guarantee much about the global distribution of names,
	// but we should make sure that we're not always using the same 5
	// names. Run up to 100 times before failing.
	//
	// NB: The odds of this flaking are extremely low. 92 choose 5 gives
	// 4,9177,128 unique combinations. After 100 shuffles, the probability of
	// seeing the same combination is astronomically low.
	for i := 0; i < 100; i++ {
		if len(seen) > nPartNames {
			return
		}
		runOneRound()
	}

	if len(seen) <= nPartNames {
		t.Errorf("only saw %d names after calling randPartName 100 times", nPartNames)
	}
}
