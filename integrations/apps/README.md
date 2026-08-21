# Reachy example apps

Trusted **robot-apps** for Device → Plugins. Each folder is a plugin
(`plugin.json` + `main.py` + `SKILL.md`). Skills in `skills/` are separate —
the brain reads those without an install.

Install from Device → Plugins (trusted list) or:

```bash
curl -X POST http://127.0.0.1:5000/api/plugin/install \
  -H "Content-Type: application/json" \
  -d '{"url":"https://github.com/K95M65/kestrel.git","subdir":"integrations/apps/dance"}'
```

Then `POST /api/plugin/dance/start`. Cameraman stays running until `/stop`.

| App | What it does |
|---|---|
| `dance` | Groove. Set `DANCE_MUSIC` for a YouTube search, or leave empty. |
| `emotions` | Spoken tour of built-in faces. |
| `cameraman` | Track a face until stopped. |
| `asl-teacher` | Five phrases with head/body (no hands). |
