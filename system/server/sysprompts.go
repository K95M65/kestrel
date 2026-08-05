package server

import "go.autonomous.ai/os/system/lib/i18n"

// System-originated prompts sent to the OpenClaw agent. Kept separate from
// server.go so they can be translated without touching boot wiring, and so
// future system messages (skill watcher updates, wellbeing nudges, …) have
// an obvious home next to the wake greeting.

// wakeGreetingPrompt is the system message fired right after the voice
// pipeline becomes ready. SOUL.md already tells the agent to mirror the
// owner's language, but an English prompt still primes English replies for
// the very first turn — so emit the prompt itself in the owner's language.
// Empty / unknown lang → English. Language is read from lib/i18n at call
// time, so caller must i18n.SetConfig before invoking.
//
// The prompt being written in the owner's language is only an IMPLICIT
// signal, and it loses to an explicit one: the agent runtime (openclaw /
// hermes / …) carries session memory that may still hold the language from
// before an stt_language switch. So the prompt also names the language
// outright and tells the agent to ignore the earlier one, plus carries the
// machine-readable [context: current_language=X] tag (same injection the
// passive-sensing path uses in lib/sensingmsg for text-less events).
func wakeGreetingPrompt() string {
	switch i18n.Lang() {
	case i18n.LangVI:
		return "[system] Bạn vừa thức dậy. Chào hỏi chủ nhân ngắn gọn. " +
			"Trả lời bằng tiếng Việt, bỏ qua ngôn ngữ của các lượt trước." +
			i18n.LangContextTag()
	case i18n.LangZhCN:
		return "[system] 你刚刚醒来，请简短地问候一下主人。" +
			"请用简体中文回复，忽略之前对话使用的语言。" +
			i18n.LangContextTag()
	case i18n.LangZhTW:
		return "[system] 你剛剛醒來，請簡短地問候一下主人。" +
			"請用繁體中文回覆，忽略之前對話使用的語言。" +
			i18n.LangContextTag()
	}
	return "[system] You just woke up. Greet the user briefly. " +
		"Reply in English, ignoring whatever language earlier turns used." +
		i18n.LangContextTag()
}
