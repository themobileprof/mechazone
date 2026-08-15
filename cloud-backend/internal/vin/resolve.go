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
		return dec, nil
	}
	fb, fbErr := r.Fallbacks.Decode(ctx, vin)
	if fbErr != nil {
		if r.Log != nil {
			r.Log.Warn("vin fallback", "err", fbErr, "vin", vin)
		}
		if err == nil {
			return dec, nil
		}
		return Decode{}, fbErr
	}
	return fb, nil
}
