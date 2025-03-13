package hostinfo

import (
	"context"
	"strconv"
	"testing"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostInfo(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p, err := New(cfg, testRegistry, nil)
	require.NoError(t, err)
	require.Equal(t, p.Name(), Name)
	defer p.Shutdown(context.TODO())

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			test.MakeBatch(10, nil),
			test.MakeBatch(10, nil),
		},
	}

	for i, b := range req.Batches {
		b.Resource.Attributes = append(b.Resource.Attributes, []*common_v1.KeyValue{
			{Key: "host.id", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
		}...)
	}

	p.PushSpans(context.Background(), req)

	lbls0 := labels.FromMap(map[string]string{
		hostIdentifierAttr: "test0",
	})
	assert.Equal(t, 1.0, testRegistry.Query(hostInfoMetric, lbls0))

	lbls1 := labels.FromMap(map[string]string{
		hostIdentifierAttr: "test1",
	})
	assert.Equal(t, 1.0, testRegistry.Query(hostInfoMetric, lbls1))
}
