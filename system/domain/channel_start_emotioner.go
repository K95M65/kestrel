package domain

// ChannelStartEmotioner is implemented by runtimes whose gateway owns channel I/O
// (hermes, picoclaw), so their native Telegram/Discord turns — which never pass
// through sendChat — still get the "thinking" emotion-acknowledge. The shared
// ChannelTurn handler type-asserts the active gateway to this interface on
// agent:start; runtimes that drive the ack elsewhere (openclaw, via its TS hook)
// don't implement it and are unaffected.
type ChannelStartEmotioner interface {
	// FireChannelStartEmotion fires the per-turn "thinking" ack for a channel turn
	// about to be answered by the gateway. message is the inbound user text; runID
	// is the channel-hook run id.
	FireChannelStartEmotion(message, runID string)
}
