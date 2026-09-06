// Package vin decodes a 17-character VIN: NHTSA vPIC first, then optional paid fallbacks.
// Results are meant to be cached in vin_decode_cache; callers must not re-query a hit.
package vin

import (
	"context"
	"log/slog"
)

type Resolver struct {
	VPIC      *Client
	Fallbacks *Fallbacks
	Log       *slog.Logger
}

// Decode tries vPIC, then optional paid fallbacks. Callers must cache the result.
func (r *Resolver) Decode(ctx context.Context, vin string) (Decode, error) {
	dec, err := r.VPIC.Decode(ctx, vin)
	if err == nil && !dec.Empty {
		return dec, nil
	}
	if err != nil && r.Log != nil {
		r.Log.Warn("vpic", "err", err, "vin", vin)
	}
	if !r.Fallbacks.Enabled() {
		if err != nil {
			return Decode{}, err
		}
		EnrichEmpty(&dec, ReadPlate(vin))
		return dec, nil
	}
	fb, fbErr := r.Fallbacks.Decode(ctx, vin)
	if fbErr != nil {
		if r.Log != nil {
			r.Log.Warn("vin fallback", "err", fbErr, "vin", vin)
		}
		if err == nil {
			EnrichEmpty(&dec, ReadPlate(vin))
			return dec, nil
		}
		return Decode{}, fbErr
	}
	if fb.Empty {
		EnrichEmpty(&fb, ReadPlate(vin))
	}
	return fb, nil
}
