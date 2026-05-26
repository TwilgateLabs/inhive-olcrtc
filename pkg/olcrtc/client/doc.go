// Package client implements the local side of the olcrtc tunnel.
//
// This package was promoted from internal/client to pkg/olcrtc/client in the
// InHive fork (TwilgateLabs/inhive-olcrtc) so that downstream embedders can
// build a sing-box / Xray-style outbound on top of the protocol's full
// client implementation (encryption, smux multiplexing, handshake, reconnect),
// rather than the simpler single-stream pkg/olcrtc.Session.
//
// # Quickstart
//
//	import (
//	    "context"
//	    "github.com/TwilgateLabs/inhive-olcrtc/pkg/olcrtc/client"
//	    _ "github.com/TwilgateLabs/inhive-olcrtc/internal/transport/datachannel" // register transport
//	    _ "github.com/TwilgateLabs/inhive-olcrtc/internal/auth/jitsi"           // register auth
//	)
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	err := client.Run(ctx, client.Config{
//	    Transport: "datachannel",
//	    Carrier:   "jitsi",
//	    RoomURL:   "https://meet1.arbitr.ru/myroom",
//	    KeyHex:    "<64-char hex>", // pre-shared with server
//	    ChannelID: "<UUID>",        // device identifier
//	    DNSServer: "9.9.9.9:53",    // Quad9 DoH recommended
//	    SocksAddr: "127.0.0.1:1080", // local SOCKS5 exposed by tunnel
//	})
//	// client.Run blocks until ctx.Done() or unrecoverable error.
//
// # Matching server
//
// The peer-side server lives in pkg/olcrtc/tunnel (see tunnel.Server).
// Client and server MUST share KeyHex (32 bytes hex-encoded) and RoomURL.
//
// # Pre-Phase-3 hardening notes (InHive-specific)
//
// When embedding in a VPN client outbound, enforce in your wrapper:
//   - Transport pinned to "datachannel" (SEC-2: avoid VP8/H.264 parser surface)
//   - DNSServer pinned to a DoH resolver (SEC-3: telemetry beacon defense)
//   - KeyHex validated as exactly 64 hex chars
//   - SocksAddr bound to 127.0.0.1 with random unguessable creds if exposed
//
// See InHive's memory/security_mitigations_olcrtc_pending.md for the
// canonical pre-Phase-3 checklist.
package client
