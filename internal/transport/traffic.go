package transport

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

var ErrTrafficPayloadTooLarge = errors.New("traffic payload exceeds max_payload_size")

type trafficTransport struct {
	inner          Transport
	maxPayloadSize int
	minDelay       time.Duration
	maxDelay       time.Duration
	sendMu         sync.Mutex
}

// WithTraffic wraps tr with optional payload caps and send pacing.
func WithTraffic(tr Transport, cfg TrafficConfig) Transport {
	if tr == nil {
		return nil
	}
	cfg = effectiveTrafficConfig(tr.Features(), cfg)
	if cfg.MaxPayloadSize <= 0 && cfg.MinDelay <= 0 && cfg.MaxDelay <= 0 {
		return tr
	}
	return &trafficTransport{
		inner:          tr,
		maxPayloadSize: cfg.MaxPayloadSize,
		minDelay:       cfg.MinDelay,
		maxDelay:       cfg.MaxDelay,
	}
}

func effectiveTrafficConfig(features Features, cfg TrafficConfig) TrafficConfig {
	if cfg.MaxPayloadSize > 0 && features.MaxPayloadSize > 0 && features.MaxPayloadSize < cfg.MaxPayloadSize {
		cfg.MaxPayloadSize = features.MaxPayloadSize
	}
	return cfg
}

func (t *trafficTransport) Connect(ctx context.Context) error { return t.inner.Connect(ctx) }

func (t *trafficTransport) Send(data []byte) error {
	t.sendMu.Lock()
	defer t.sendMu.Unlock()
	if t.maxPayloadSize > 0 && len(data) > t.maxPayloadSize {
		return fmt.Errorf("%w: size=%d max=%d", ErrTrafficPayloadTooLarge, len(data), t.maxPayloadSize)
	}
	if delay := t.nextDelay(); delay > 0 {
		time.Sleep(delay)
	}
	return t.inner.Send(data)
}

func (t *trafficTransport) Close() error { return t.inner.Close() }

func (t *trafficTransport) SetReconnectCallback(cb func()) { t.inner.SetReconnectCallback(cb) }

func (t *trafficTransport) SetShouldReconnect(fn func() bool) { t.inner.SetShouldReconnect(fn) }

func (t *trafficTransport) SetEndedCallback(cb func(string)) { t.inner.SetEndedCallback(cb) }

func (t *trafficTransport) WatchConnection(ctx context.Context) { t.inner.WatchConnection(ctx) }

func (t *trafficTransport) CanSend() bool { return t.inner.CanSend() }

func (t *trafficTransport) Features() Features {
	features := t.inner.Features()
	if t.maxPayloadSize > 0 &&
		(features.MaxPayloadSize == 0 || t.maxPayloadSize < features.MaxPayloadSize) {
		features.MaxPayloadSize = t.maxPayloadSize
	}
	return features
}

func (t *trafficTransport) nextDelay() time.Duration {
	if t.maxDelay <= 0 && t.minDelay <= 0 {
		return 0
	}
	minDelay := t.minDelay
	maxDelay := t.maxDelay
	if maxDelay <= minDelay {
		return minDelay
	}
	return minDelay + time.Duration(rand.Int64N(int64(maxDelay-minDelay))) //nolint:gosec,lll // G404: non-cryptographic pacing jitter
}
