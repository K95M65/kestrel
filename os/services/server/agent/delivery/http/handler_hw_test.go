package http

import "testing"

func TestExtractHWCallsCanonical(t *testing.T) {
	calls, rest := extractHWCalls(`[HW:/led/off:{}] LEDs off.`)
	if len(calls) != 1 || calls[0].path != "/led/off" || calls[0].body != "{}" {
		t.Fatalf("canonical marker not parsed: %+v", calls)
	}
	if rest != "LEDs off." {
		t.Fatalf("stripped text = %q", rest)
	}
}

func TestExtractHWCallsMarkdownLinkForm(t *testing.T) {
	// Markdown-link form emitted by some LLMs instead of the canonical marker.
	calls, rest := extractHWCalls(`[Tắt đèn liền đây!](HW:/led/off:{})`)
	if len(calls) != 1 || calls[0].path != "/led/off" || calls[0].body != "{}" {
		t.Fatalf("link-form marker not parsed: %+v", calls)
	}
	if rest != "Tắt đèn liền đây!" {
		t.Fatalf("label lost, stripped text = %q", rest)
	}
}

func TestExtractHWCallsLinkFormWithBody(t *testing.T) {
	calls, rest := extractHWCalls(`[Đèn đỏ nè](HW:/led/solid:{"color":[255,0,0]}) xong rồi.`)
	if len(calls) != 1 || calls[0].path != "/led/solid" || calls[0].body != `{"color":[255,0,0]}` {
		t.Fatalf("link-form body not parsed: %+v", calls)
	}
	if rest != "Đèn đỏ nè xong rồi." {
		t.Fatalf("stripped text = %q", rest)
	}
}

func TestExtractHWCallsLinkFormBodyless(t *testing.T) {
	calls, _ := extractHWCalls(`[Dừng hiệu ứng](HW:/led/effect/stop)`)
	if len(calls) != 1 || calls[0].path != "/led/effect/stop" || calls[0].body != "{}" {
		t.Fatalf("bodyless link-form not parsed: %+v", calls)
	}
}

func TestExtractHWCallsPlainLinkUntouched(t *testing.T) {
	calls, rest := extractHWCalls(`Xem [tài liệu](https://example.com) nhé.`)
	if len(calls) != 0 {
		t.Fatalf("plain markdown link misparsed as HW call: %+v", calls)
	}
	if rest != `Xem [tài liệu](https://example.com) nhé.` {
		t.Fatalf("plain link mangled: %q", rest)
	}
}

func TestExtractLeadingHWCallsLinkForm(t *testing.T) {
	// A reply opening with a link-form marker must still fire at stream time.
	calls := extractLeadingHWCalls(`[Tắt đèn liền đây!](HW:/led/off:{})`)
	if len(calls) != 1 || calls[0].path != "/led/off" || calls[0].body != "{}" {
		t.Fatalf("leading link-form not detected: %+v", calls)
	}
}
