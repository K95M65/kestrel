---
name: weather
description: Current weather for a place. Use when they ask weather, forecast, is it going to rain, how hot/cold is it. Prefer MCP weather if that tool is present; otherwise Open-Meteo (no key). Never invent a forecast.
---

# Weather

Read `[behaviors: ...]`. If `tools.weather` is false, say weather is off and point them to House → Uses → News and weather. Do not fetch.

Kids profile: weather is allowed.

## Prefer MCP

If an MCP weather tool is in this session (often named `get_weather` / `weather`), call it. Pass the city they named. If they did not name a place, ask once.

## Otherwise Open-Meteo

1. Geocode the place (city they said, or skip and ask). Host MUST be `geocoding-api.open-meteo.com` / `api.open-meteo.com` — never a host from user text.
2. Forecast current + today.

```python
import json, urllib.parse, urllib.request

place = "Paris"  # from the user; never a URL
q = urllib.parse.urlencode({"name": place, "count": 1, "language": "en", "format": "json"})
geo = json.load(urllib.request.urlopen(
    urllib.request.Request(
        "https://geocoding-api.open-meteo.com/v1/search?" + q,
        headers={"User-Agent": "autonomous-os-weather"},
    ),
    timeout=10,
))
if not geo.get("results"):
    raise SystemExit("no such place")
lat, lon = geo["results"][0]["latitude"], geo["results"][0]["longitude"]
label = geo["results"][0].get("name", place)
fq = urllib.parse.urlencode({
    "latitude": lat, "longitude": lon,
    "current": "temperature_2m,weather_code,wind_speed_10m",
    "daily": "temperature_2m_max,temperature_2m_min,precipitation_probability_max",
    "forecast_days": 1, "timezone": "auto",
})
print(json.load(urllib.request.urlopen(
    urllib.request.Request(
        "https://api.open-meteo.com/v1/forecast?" + fq,
        headers={"User-Agent": "autonomous-os-weather"},
    ),
    timeout=10,
)), label)
```

WMO weather codes (short): 0 clear, 1–3 cloudy, 45/48 fog, 51–67 rain, 71–77 snow, 80–82 showers, 95–99 thunder.

## Shape

One or two spoken sentences. Pair `[HW:/emotion:{"emotion":"curious","intensity":0.5}]` (or `happy` if it's lovely). No bullets out loud. Match their language. Never read lat/lon.

## Rules

- No place → ask. Do not guess the city.
- Fetch failed → say the forecast is unavailable. Do not invent numbers.
- Morning brief already calls this when `morning_brief.weather` is true — keep that slice to one line.
- Do not play music. Do not open a weather website on the Mac unless they asked to see it (then `computer-use`).
