---
name: radio
description: Leave a radio stream on the speaker until they say stop. Use when they say radio, put something on, leave music playing.
---

# Radio

Kids: allowed (speaker only). If `[behaviors] radio` is false, say you are sitting this one out.

Start (leave it on):
```
[HW:/plugin/start:{"name":"radio"}]
```

Stop:
```
[HW:/plugin/stop:{"name":"radio"}]
```

If start fails, tell them to Install **Radio** under Device → Plugins. Do not curl plugin URLs. One short spoken line.
