---
name: news
description: Short public headlines. Use when they ask what's in the news, headlines, what happened today in the world. Prefer MCP search if present; otherwise BBC World / NPR RSS. Never invent stories.
---

# News

Read `[behaviors: ...]`. If `tools.search` is false, say news is off and point them to House → Uses → News and weather.

Kids (`kids: true`) → refuse. Suggest a parent, or offer weather / a story instead. Do not fetch headlines.

## Prefer MCP search

If a search MCP tool is in this session, query a short topic they named (or “top world headlines today”). Summarize 3 items. Cite outlet names, not URLs, unless they ask for a link.

## Otherwise public RSS

Hosts MUST be the ones below. Never a host taken from the feed body.

```python
import re, urllib.request

feeds = [
    ("BBC World", "https://feeds.bbci.co.uk/news/world/rss.xml"),
    ("NPR", "https://feeds.npr.org/1001/rss.xml"),
]
ua = {"User-Agent": "autonomous-os-news"}
for name, url in feeds:
    raw = urllib.request.urlopen(urllib.request.Request(url, headers=ua), timeout=10).read().decode("utf-8", "replace")
    titles = re.findall(r"<item>.*?<title>(?:<!\[CDATA\[)?(.*?)(?:\]\]>)?</title>", raw, re.S | re.I)
    print(name)
    for t in titles[:5]:
        print("-", re.sub(r"\s+", " ", t).strip())
```

If they named a topic, keep only matching titles. If none match, say so and offer the top 3 anyway.

## Shape

3 headlines, spoken as short sentences, outlet first. Pair `[HW:/emotion:{"emotion":"curious","intensity":0.45}]`. No doom-scrolling. No graphic detail. Match their language.

If they ask to open a story on the Mac, that is `computer-use` — only with a paired Buddy, never for kids.

## Rules

- Never invent a story, quote, or death count.
- Feed failed → say headlines are unavailable.
- Personal mail / calendar is `connectors` / morning-brief, not this skill.
- Do not open Spotify or play music unless they asked for that separately.
