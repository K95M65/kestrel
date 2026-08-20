const TTS_PROVIDER_LABELS: Record<string, string> = {
  elevenlabs: "ElevenLabs",
  openai: "OpenAI",
  grok: "xAI Grok",
  xai: "xAI",
  piper: "Piper (on device)",
  kokoro: "Kokoro",
  edge: "Microsoft Edge",
  google: "Google",
  azure: "Azure",
};

export function ttsProviderLabel(id: string): string {
  return TTS_PROVIDER_LABELS[id.toLowerCase()] ?? id;
}
