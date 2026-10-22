package license

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

type restrictiveLicense struct{ fleet.LicenseInfo }

func (restrictiveLicense) IsPremium() bool               { return false }
func (restrictiveLicense) IsAllowDisableTelemetry() bool { return false }

func TestIsPremium(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"no license", context.Background(), true},
		{"restrictive checker", NewContext(context.Background(), &restrictiveLicense{}), true},
		{"nil license", NewContext(context.Background(), (*fleet.LicenseInfo)(nil)), true},
		{"expired license", NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Unix(0, 0)}), true},
		{"free license", NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree}), true},
		{"premium license", NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}), true},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsPremium(c.ctx)
			require.Equal(t, c.want, got)
			require.True(t, IsAllowDisableTelemetry(c.ctx))
		})
	}
}
