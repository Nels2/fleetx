package fleet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLicenseIndependentEntitlements(t *testing.T) {
	for _, tier := range []string{TierFree, TierPremium, ""} {
		t.Run(tier, func(t *testing.T) {
			for _, expiration := range []time.Time{time.Time{}, time.Unix(0, 0), time.Now().Add(24 * time.Hour)} {
				lic := &LicenseInfo{Tier: tier, Expiration: expiration, DeviceCount: 1}
				require.True(t, lic.IsPremium())
				require.False(t, lic.IsExpired())
				require.True(t, lic.IsAllowDisableTelemetry())
				require.Equal(t, tier, lic.Tier)
				require.Equal(t, expiration, lic.Expiration)
				require.Equal(t, 1, lic.DeviceCount)
			}
		})
	}
	var lic *LicenseInfo
	require.True(t, lic.IsPremium())
	require.False(t, lic.IsExpired())
}
