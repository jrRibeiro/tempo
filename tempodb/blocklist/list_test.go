package blocklist

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

const testTenantID = "test"

func TestApplyPollResults(t *testing.T) {
	tests := []struct {
		name            string
		metas           PerTenant
		compacted       PerTenantCompacted
		expectedTenants []string
	}{
		{
			name:            "all nil",
			expectedTenants: []string{},
		},
		{
			name: "meta only",
			metas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				"test2": []*backend.BlockMeta{
					{
						BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expectedTenants: []string{"test", "test2"},
		},
		{
			name: "compacted meta only",
			compacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
						},
					},
				},
			},
			expectedTenants: []string{},
		},
		{
			name: "all",
			metas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				"blerg": []*backend.BlockMeta{
					{
						BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			compacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
				},
				"test2": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000001"),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.MustParse("00000000-0000-0000-0000-000000000002"),
						},
					},
				},
			},
			expectedTenants: []string{"blerg", "test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := New()
			l.ApplyPollResults(tc.metas, tc.compacted)

			actualTenants := l.Tenants()
			sort.Slice(actualTenants, func(i, j int) bool { return actualTenants[i] < actualTenants[j] })
			assert.Equal(t, tc.expectedTenants, actualTenants)
			for tenant, expected := range tc.metas {
				actual := l.Metas(tenant)
				assert.Equal(t, expected, actual)
			}
			for tenant, expected := range tc.compacted {
				actual := l.CompactedMetas(tenant)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	var (
		_1  = meta("00000000-0000-0000-0000-000000000001")
		_2  = meta("00000000-0000-0000-0000-000000000002")
		_3  = meta("00000000-0000-0000-0000-000000000003")
		_2c = compactedMeta("00000000-0000-0000-0000-000000000002")
		_3c = compactedMeta("00000000-0000-0000-0000-000000000003")
	)

	tests := []struct {
		name     string
		existing []*backend.BlockMeta
		add      []*backend.BlockMeta
		remove   []*backend.BlockMeta
		addC     []*backend.CompactedBlockMeta
		removeC  []*backend.CompactedBlockMeta
		expected []*backend.BlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			remove:   nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add:      []*backend.BlockMeta{_1},
			remove:   nil,
			expected: []*backend.BlockMeta{_1},
		},
		{
			name:     "add to existing",
			existing: []*backend.BlockMeta{_1},
			add:      []*backend.BlockMeta{_2},
			remove:   nil,
			expected: []*backend.BlockMeta{_1, _2},
		},
		{
			name:     "remove from nil",
			existing: nil,
			add:      nil,
			remove:   []*backend.BlockMeta{_2},
			expected: nil,
		},
		{
			name:     "remove nil",
			existing: []*backend.BlockMeta{_2},
			add:      nil,
			remove:   nil,
			expected: []*backend.BlockMeta{_2},
		},
		{
			name:     "remove existing",
			existing: []*backend.BlockMeta{_1, _2},
			add:      nil,
			remove:   []*backend.BlockMeta{_1},
			expected: []*backend.BlockMeta{_2},
		},
		{
			name:     "remove no match",
			existing: []*backend.BlockMeta{_1},
			add:      nil,
			remove:   []*backend.BlockMeta{_2},
			expected: []*backend.BlockMeta{_1},
		},
		{
			name:     "add and remove",
			existing: []*backend.BlockMeta{_1, _2},
			add:      []*backend.BlockMeta{_3},
			remove:   []*backend.BlockMeta{_2},
			expected: []*backend.BlockMeta{_1, _3},
		},
		{
			name:     "add already exists",
			existing: []*backend.BlockMeta{_1},
			add:      []*backend.BlockMeta{_1, _2},
			remove:   nil,
			expected: []*backend.BlockMeta{_1, _2},
		},
		{
			name:     "not added if also removed",
			existing: []*backend.BlockMeta{_1},
			add:      []*backend.BlockMeta{_2},
			remove:   []*backend.BlockMeta{_2},
			expected: []*backend.BlockMeta{_1},
		},
		{
			name:     "not added if also compacted",
			existing: []*backend.BlockMeta{_1},
			add:      []*backend.BlockMeta{_2, _3},
			addC:     []*backend.CompactedBlockMeta{_2c},
			removeC:  []*backend.CompactedBlockMeta{_3c},
			expected: []*backend.BlockMeta{_1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.metas[testTenantID] = tt.existing
			l.Update(testTenantID, tt.add, tt.remove, tt.addC, tt.removeC)

			require.Equal(t, len(tt.expected), len(l.metas[testTenantID]))
			require.ElementsMatch(t, tt.expected, l.metas[testTenantID])
		})
	}
}

func TestUpdateCompacted(t *testing.T) {
	var (
		_1 = compactedMeta("00000000-0000-0000-0000-000000000001")
		_2 = compactedMeta("00000000-0000-0000-0000-000000000002")
		_3 = compactedMeta("00000000-0000-0000-0000-000000000003")
	)

	tests := []struct {
		name     string
		existing []*backend.CompactedBlockMeta
		add      []*backend.CompactedBlockMeta
		remove   []*backend.CompactedBlockMeta
		expected []*backend.CompactedBlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add:      []*backend.CompactedBlockMeta{_1},
			expected: []*backend.CompactedBlockMeta{_1},
		},
		{
			name:     "add to existing",
			existing: []*backend.CompactedBlockMeta{_1},
			add:      []*backend.CompactedBlockMeta{_2},
			expected: []*backend.CompactedBlockMeta{_1, _2},
		},
		{
			name:     "add already exists",
			existing: []*backend.CompactedBlockMeta{_1},
			add:      []*backend.CompactedBlockMeta{_1, _2},
			expected: []*backend.CompactedBlockMeta{_1, _2},
		},
		{
			name:     "add and remove",
			existing: []*backend.CompactedBlockMeta{_1, _2},
			add:      []*backend.CompactedBlockMeta{_3},
			remove:   []*backend.CompactedBlockMeta{_2},
			expected: []*backend.CompactedBlockMeta{_1, _3},
		},
		{
			name:     "not added if also removed",
			existing: []*backend.CompactedBlockMeta{_1},
			add:      []*backend.CompactedBlockMeta{_2},
			remove:   []*backend.CompactedBlockMeta{_2},
			expected: []*backend.CompactedBlockMeta{_1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()

			l.compactedMetas[testTenantID] = tt.existing
			l.Update(testTenantID, nil, nil, tt.add, tt.remove)

			assert.Equal(t, len(tt.expected), len(l.compactedMetas[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockID, l.compactedMetas[testTenantID][i].BlockID)
			}
		})
	}
}

func TestUpdatesSaved(t *testing.T) {
	// unlike most tests these are applied serially to the same list object and the expected
	// results are cumulative across all tests

	one := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	two := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	oneOhOne := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	oneOhTwo := uuid.MustParse("10000000-0000-0000-0000-000000000002")

	tests := []struct {
		applyMetas     PerTenant
		applyCompacted PerTenantCompacted
		updateTenant   string
		addMetas       []*backend.BlockMeta
		removeMetas    []*backend.BlockMeta
		addCompacted   []*backend.CompactedBlockMeta

		expectedTenants   []string
		expectedMetas     PerTenant
		expectedCompacted PerTenantCompacted
	}{
		// STEP 1: apply a normal polling data and updates
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.UUID(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
				},
			},
			updateTenant: "test",
			addMetas: []*backend.BlockMeta{
				{
					BlockID: backend.UUID(one),
				},
				{
					BlockID: backend.UUID(two),
				},
			},
			removeMetas: []*backend.BlockMeta{
				{
					BlockID: backend.UUID(one),
				},
			},
			addCompacted: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: backend.UUID(oneOhTwo),
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.UUID(two),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhTwo),
						},
					},
				},
			},
		},
		// STEP 2: same polling apply, no update! but expect the same results
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.UUID(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					// Even though we have just appled one, it was removed in the previous step, and we we expect not to find it here.
					// {
					// 	BlockID: one,
					// },
					{
						BlockID: backend.UUID(two),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhTwo),
						},
					},
				},
			},
		},
		// STEP 3: same polling apply, no update! but this time the update doesn't impact results
		{
			applyMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.UUID(one),
					},
				},
			},
			applyCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
				},
			},
			expectedTenants: []string{"test"},
			expectedMetas: PerTenant{
				"test": []*backend.BlockMeta{
					{
						BlockID: backend.UUID(one),
					},
				},
			},
			expectedCompacted: PerTenantCompacted{
				"test": []*backend.CompactedBlockMeta{
					{
						BlockMeta: backend.BlockMeta{
							BlockID: backend.UUID(oneOhOne),
						},
					},
				},
			},
		},
	}

	l := New()
	for i, tc := range tests {
		t.Logf("step %d", i+1)

		l.ApplyPollResults(tc.applyMetas, tc.applyCompacted)
		if tc.updateTenant != "" {
			l.Update(tc.updateTenant, tc.addMetas, tc.removeMetas, tc.addCompacted, nil)
		}

		actualTenants := l.Tenants()
		actualMetas := l.metas
		actualCompacted := l.compactedMetas

		sort.Slice(actualTenants, func(i, j int) bool { return actualTenants[i] < actualTenants[j] })
		assert.Equal(t, tc.expectedTenants, actualTenants)
		assert.Equal(t, tc.expectedMetas, actualMetas)

		require.Equal(t, len(tc.expectedCompacted), len(actualCompacted), "expectedCompacted: %+v, actualCompacted: %+v", tc.expectedCompacted, actualCompacted)
		assert.Equal(t, tc.expectedCompacted, actualCompacted)
	}
}

func BenchmarkUpdate(b *testing.B) {
	var (
		l         = New()
		numBlocks = 100000 // Realistic number
		existing  = make([]*backend.BlockMeta, 0, numBlocks)
		add       = []*backend.BlockMeta{
			meta("00000000-0000-0000-0000-000000000001"),
			meta("00000000-0000-0000-0000-000000000002"),
		}
		remove = []*backend.BlockMeta{
			meta("00000000-0000-0000-0000-000000000003"),
			meta("00000000-0000-0000-0000-000000000004"),
		}
		numCompacted = 1000 // Realistic number
		compacted    = make([]*backend.CompactedBlockMeta, 0, numCompacted)
		compactedAdd = []*backend.CompactedBlockMeta{
			compactedMeta("00000000-0000-0000-0000-000000000005"),
			compactedMeta("00000000-0000-0000-0000-000000000006"),
		}
		compactedRemove = []*backend.CompactedBlockMeta{
			compactedMeta("00000000-0000-0000-0000-000000000007"),
			compactedMeta("00000000-0000-0000-0000-000000000008"),
		}
	)

	for i := 0; i < numBlocks; i++ {
		existing = append(existing, meta(uuid.NewString()))
	}
	for i := 0; i < numCompacted; i++ {
		compacted = append(compacted, compactedMeta(uuid.NewString()))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.metas[testTenantID] = existing
		l.compactedMetas[testTenantID] = compacted
		l.Update(testTenantID, add, remove, compactedAdd, compactedRemove)
	}
}

func meta(id string) *backend.BlockMeta {
	return &backend.BlockMeta{
		BlockID: backend.MustParse(id),
	}
}

func compactedMeta(id string) *backend.CompactedBlockMeta {
	return &backend.CompactedBlockMeta{
		BlockMeta: backend.BlockMeta{
			BlockID: backend.MustParse(id),
		},
	}
}
