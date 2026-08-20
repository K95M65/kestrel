/** Strip hardware markers and model-control leaks from Talk / guide replies.
 *
 *  HW_LINK_RE / HW_MARKER_RE mirror handler_hw.go (hwLinkRe / hwMarkerRe)
 *  exactly — never looser. A variant the executor will not fire must stay
 *  visible as raw text instead of being scrubbed into a confident-looking
 *  label.
 *
 *  Everything after that is a leak filter (HAL strip_rt_markers, runtime
 *  stripForChannel, plus slash-command / workspace-doc leaks Talk showed
 *  on 0.1.36). Those are not executor grammar.
 */

const HW_LINK_RE =
  /\[([^\]]*)\]\(\s*HW:\s*(?:\/[^(){:\s]+(?::[^(){:\s]+)*)(?::\{[^}]*\})?:?\s*\)/gi;
const HW_MARKER_RE = /\[HW:\/[^{\]]*(?:\{[^}]*\})?\]/g;

// Delivery + emotion tags the model dumps into the bubble. HAL RT_MARKER_RE
// plus the expression names the emotion skill uses as [excited] / [curious].
const BRACKET_TAG_RE =
  /\[(?:laughs?|sighs?|whisper(?:s)?|gasp(?:s)?|gulp(?:s)?|nervous(?:ly)?|excited|frustrated|sorrowful|calm|cheerfully|playfully|quietly|deadpan|flatly|dramatic tone|resigned tone|tired|sad|happy|curious|caring|thinking|thinks|thought|thoughtful|pondering|ponders|reasoning|idle|shy|shock|confused|sleepy|greeting|goodbye|acknowledge|laugh|listening|music_chill|music_strong|chuckle|light chuckle|giggle|big laugh|breathes|clears throat|pause(?:s)?|hesitates|stammers)[^\]]*\]/gi;

const BACKTICK_TAG_RE = /`\[(?:[^\]]*)\]`/g;

// `/emotion name="happy" intensity="0.8"` and the shorter `/emotion` HAL strips.
const SLASH_CMD_RE =
  /\/(?:emotion|servo|led|skills|scene)(?:\s+\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s]+))*/gi;

const EMOTION_CTX_RE = /emotion_(?:user|model)\s*:\s*\S+/gi;
const INTENSITY_TOKEN_RE = /\bintensity\s*[:=]\s*[0-9.]+/gi;
const NO_REPLY_RE = /\bNO_REPLY\b/g;

const INTERNAL_DOC_NAME =
  /^(?:MEMORY|SKILL|IDENTITY|SOUL|USER|HEARTBEAT|AGENTS|TOOLS|BOOT|SAFETY|ROBOT)\.md$/i;

const INTERNAL_MD_LINK_RE = new RegExp(
  String.raw`\[(?:[^\]]*)\]\(\s*(?:[^)\s]*\/)?(?:MEMORY|SKILL|IDENTITY|SOUL|USER|HEARTBEAT|AGENTS|TOOLS|BOOT|SAFETY|ROBOT)\.md\s*\)`,
  "gi",
);

const WORKSPACE_MD_PATH_RE =
  /\/(?:root\/\.[a-z0-9_-]+\/(?:media|workspace)|tmp)\/[^\s"'`)<>\]]+\.md\b/gi;

export function isInternalAgentDoc(path: string): boolean {
  const name = path.slice(path.lastIndexOf("/") + 1);
  return INTERNAL_DOC_NAME.test(name);
}

export function stripChatMarkers(text: string): string {
  if (!text) return text;
  return text
    .replace(HW_LINK_RE, (_m, label: string) => (/^hw:/i.test(label) ? "" : label))
    .replace(HW_MARKER_RE, "")
    .replace(BRACKET_TAG_RE, "")
    .replace(BACKTICK_TAG_RE, "")
    .replace(SLASH_CMD_RE, "")
    .replace(EMOTION_CTX_RE, "")
    .replace(INTENSITY_TOKEN_RE, "")
    .replace(NO_REPLY_RE, "")
    .replace(INTERNAL_MD_LINK_RE, "")
    .replace(WORKSPACE_MD_PATH_RE, "")
    .replace(/[^\S\n]{2,}/g, " ")
    .replace(/[ \t]+\n/g, "\n")
    .replace(/ +([.,;:!?])/g, "$1")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
