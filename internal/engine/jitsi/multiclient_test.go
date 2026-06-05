package jitsi

import (
	"context"
	"testing"

	"github.com/openlibrecommunity/olcrtc/internal/engine"
)

// newClientSession builds a client-side (cnc) Session: onData is wired,
// onPeerData stays nil, so deliverBridgeMessage takes the single-peer
// latch path (not the joiner per-occupant path).
//
// RequireTargetedPeer mirrors production: client.bringUpLink hard-sets it true
// for every client link (internal/client/client.go), so the upstream
// targeted-peer guard (acceptEpochFrame, 7a6cb7f) drops untargeted peer-client
// frames. A test that omits it exercises a non-production config and will
// reproduce the old multi-client regression — see TestMultiClient_FlagOff.
func newClientSession(t *testing.T, onData func([]byte)) *Session {
	t.Helper()
	sess, err := New(context.Background(), engine.Config{
		URL:                 testHost,
		Extra:               map[string]string{credentialKeyRoom: testRoom},
		RequireTargetedPeer: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	js, ok := sess.(*Session)
	if !ok {
		t.Fatal("sess is not *Session")
	}
	js.onData = onData
	return js
}

// TestMultiClientDoesNotLatchOntoPeerClientAnnounce reproduces the multi-client
// regression ("two phones on one config, only one gets traffic").
//
// Topology: one joiner + two client (cnc) sessions share a MUC. Client A has
// already latched onto the joiner. Client B then enters and broadcasts its
// announce with receiverEpoch=0 (it has not learned the joiner's epoch yet).
// Client A sees B's broadcast. The frame is NOT for A — it is B trying to reach
// the joiner — so A must ignore it and stay latched on the joiner.
//
// Bug: frameAddressedToUs() treats receiverEpoch==0 as "addressed to us"
// (handshake escape hatch), so peerLatchAccepts re-latches A onto B and A loses
// its link to the joiner. FAIL on v0.0.5..v0.0.7, PASS on v0.0.4 (which never
// re-latches at all — but v0.0.4 instead fails the rejoin case).
func TestMultiClientDoesNotLatchOntoPeerClientAnnounce(t *testing.T) {
	const epochA, epochJ, epochB = 0xAAAA, 0xCCCC, 0xBBBB

	var received [][]byte
	js := newClientSession(t, func(b []byte) {
		received = append(received, append([]byte(nil), b...))
	})
	js.localEpoch.Store(epochA)

	// 1. Joiner sends A a real frame addressed to A → A latches onto the joiner.
	jf := makeBridgeFrameForEpoch(t, epochJ, epochA, []byte("from-joiner"))
	js.deliverBridgeMessage(makeBridgeMessageFrom("joiner", map[string]any{rawFieldKey: jf}), true)
	if p := js.peerEndpoint.Load(); p == nil || *p != "joiner" {
		t.Fatalf("setup: after joiner frame latch=%v, want joiner", p)
	}

	// 2. Peer client B broadcasts its announce (receiverEpoch=0: B doesn't know
	//    the joiner's epoch yet). A must IGNORE it and keep its joiner latch.
	bf := makeBridgeFrameForEpoch(t, epochB, 0, []byte("from-peer-client-B"))
	js.deliverBridgeMessage(makeBridgeMessageFrom("clientB", map[string]any{rawFieldKey: bf}), true)

	if p := js.peerEndpoint.Load(); p == nil || *p != "joiner" {
		t.Fatalf("REGRESSION: latch moved to %v after peer-client announce, want joiner "+
			"(client A stole onto client B → loses joiner link → one-channel)", p)
	}
	// B's payload must not have been delivered to A's data path either.
	for _, r := range received {
		if string(r) == "from-peer-client-B" {
			t.Fatal("REGRESSION: peer client B's payload leaked into client A's data path")
		}
	}
}

// TestRejoinReLatchesOntoReconnectedJoiner is the competing-requirement twin of
// the multi-client test: with RequireTargetedPeer on (production), the client
// must STILL re-latch when the *joiner* reconnects as a new MUC endpoint and
// addresses the client by epoch. This is the #9 rejoin case that v0.0.4 could
// not do (it never re-latched at all). v0.0.8 satisfies both because the joiner
// targets the client's epoch on send (jitsi.go peerEpochFor), so its post-
// reconnect frames carry receiverEpoch == our localEpoch and pass the targeted-
// peer guard — whereas a peer client's broadcast (receiverEpoch=0) does not.
func TestRejoinReLatchesOntoReconnectedJoiner(t *testing.T) {
	const epochA, epochJ1, epochJ2 = 0xAAAA, 0xCCCC, 0xDDDD

	var received [][]byte
	js := newClientSession(t, func(b []byte) {
		received = append(received, append([]byte(nil), b...))
	})
	js.localEpoch.Store(epochA)

	// 1. Joiner endpoint #1 addresses A → A latches onto "joiner1".
	j1 := makeBridgeFrameForEpoch(t, epochJ1, epochA, []byte("from-joiner1"))
	js.deliverBridgeMessage(makeBridgeMessageFrom("joiner1", map[string]any{rawFieldKey: j1}), true)
	if p := js.peerEndpoint.Load(); p == nil || *p != "joiner1" {
		t.Fatalf("setup: latch=%v, want joiner1", p)
	}

	// 2. Joiner reconnects as a NEW MUC endpoint "joiner2", still addressing A by
	//    epoch (it remembers A). A MUST re-latch onto joiner2 — losing this is
	//    the #9 rejoin bug (room lost when the peer leaves/returns).
	j2 := makeBridgeFrameForEpoch(t, epochJ2, epochA, []byte("from-joiner2"))
	js.deliverBridgeMessage(makeBridgeMessageFrom("joiner2", map[string]any{rawFieldKey: j2}), true)
	if p := js.peerEndpoint.Load(); p == nil || *p != "joiner2" {
		t.Fatalf("REJOIN BROKEN: latch=%v after joiner reconnect, want joiner2 "+
			"(targeted-peer guard must not block the joiner's epoch-addressed rejoin frame)", p)
	}
	// joiner2's payload must reach A's data path.
	got := false
	for _, r := range received {
		if string(r) == "from-joiner2" {
			got = true
		}
	}
	if !got {
		t.Fatal("REJOIN BROKEN: joiner2 payload never delivered to client data path")
	}
}
