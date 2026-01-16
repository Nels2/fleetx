package license

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestIsPremium(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"no license", context.Background(), true},
		{"free license", NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree}), true},
		{"premium license", NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}), true},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsPremium(c.ctx)
			require.Equal(t, c.want, got)
		})
	}
}
