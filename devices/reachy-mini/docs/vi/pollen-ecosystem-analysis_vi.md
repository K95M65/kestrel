# Tham Khảo Hệ Sinh Thái Pollen

Tài liệu kỹ thuật về cách hệ sinh thái Pollen Robotics / Hugging Face hoạt
động cho Reachy Mini. Bao gồm voice architecture, tool registry, app
distribution, và bài học rút ra để nâng cấp Autonomous OS.

Viết tháng 7/2026.

## Voice Architecture

Conversation app của Pollen chạy voice loop một pipeline duy nhất:

```
Mic → Silero VAD → Parakeet STT → LLM (có tools qua MCP) → Qwen3 TTS → Loa
      └──── tất cả trong 1 process, 1 WebSocket ────────────────────────┘
```

Đặc điểm chính:
- **Không chia pipeline.** LLM có full tool access (qua MCP) mỗi turn, nên
  không cần tách đường "nhanh" vs đường "thông minh".
- **VAD server-side.** Silero chạy cùng process với STT, không có network hop.
- **LLM backend thay đổi được:** llama.cpp (local 4B), vLLM, OpenAI, HF
  Inference.
- **Cascaded STT+TTS** (không phải native speech-to-speech). Chất lượng giọng
  tốt nhưng không bằng native Gemini/OpenAI Realtime.

### Bài Học

Một pipeline duy nhất có full tool access mỗi turn thì kiến trúc đơn giản hơn
so với chia đường nhanh-nhưng-hạn-chế vs chậm-nhưng-đầy-đủ. Đáng prototype
trên Reachy (greenfield, chưa có voice path legacy) để đánh giá trade-off
latency và chất lượng trước khi quyết định backport.

## Tool Registry

Conversation app có hệ thống tool ba tầng:

### Tầng 1: Built-In Tools (Local)

Ship cùng app trong `src/.../tools/`. Mỗi tool là Python class kế thừa `Tool`,
đăng ký vào dict toàn cục `ALL_TOOLS`. 17 built-in tools: `move_head`, `dance`,
`stop_dance`, `play_emotion`, `camera`, `idle_do_nothing`, `head_tracking`,
`remember`, `forget`, `go_to_sleep`, v.v.

### Tầng 2: Custom Local Tools (User Tự Viết)

File Python trong `external_tools/`. Tự động load khi
`AUTOLOAD_EXTERNAL_TOOLS=1`. Mỗi file phải expose `Tool.name` duy nhất. App báo
lỗi ngay khi có va chạm tên.

### Tầng 3: Remote MCP Tools (HF Spaces)

Quản lý bởi `tool_spaces.py` và `mcp_client.py`. Tools chạy remote trong Gradio
Spaces công khai, expose MCP tại `/gradio_api/mcp/`. Không download code về
robot — LLM gọi tools qua HTTPS.

```bash
# Cài remote tool
reachy-mini-conversation-app tool-spaces add pollen-robotics/reachy-mini-search-tool

# Liệt kê / gỡ bỏ
reachy-mini-conversation-app tool-spaces list
reachy-mini-conversation-app tool-spaces remove <owner/space>
```

**Namespacing** dùng dấu gạch dưới kép để tránh va chạm:
```
Space:  pollen-robotics/reachy-mini-search-tool
Alias:  pollen_robotics_reachy_mini_search_tool
Tool:   pollen_robotics_reachy_mini_search_tool__search_web
```

**Remote tools cài sẵn:**
- `pollen-robotics/reachy-mini-search-tool` (tìm kiếm web)
- `pollen-robotics/reachy-mini-time-tool` (thời gian/múi giờ)
- `pollen-robotics/reachy-mini-weather-tool` (thời tiết)

**Gating theo profile:** mỗi profile có `tools.txt` whitelist — tool chỉ khả
dụng nếu ID của nó có trong danh sách profile đang active.

### Bài Học

Thêm MCP client vào agent gateway sẽ cho phép Autonomous OS consume HF Space
tools cùng local `SKILL.md`. Publish vài skills thành Gradio MCP endpoints sẽ
tạo visibility trong HF ecosystem. Cả hai có thể cùng tồn tại — local skills
cho offline/tích hợp sâu, remote MCP tools cho community contributions.

## App Distribution

Hai kênh để đưa capabilities lên Reachy Mini:

### Kênh A: Apps (Hành Vi Robot Đầy Đủ)

- Python packages kế thừa `ReachyMiniApp` với method `run()`
- Tag `reachy_mini_python_app` trên HF Spaces
- Cài one-click từ dashboard hoặc REST API:
  ```bash
  curl -X POST http://localhost:8000/api/apps/install \
    -H "Content-Type: application/json" \
    -d '{"url": "https://huggingface.co/spaces/<user>/<app>"}'
  ```
- Download và chạy local dưới dạng subprocess
- Full access phần cứng robot (motion, audio, vision)
- 200+ apps từ 150+ creators tính đến giữa 2026

### Kênh B: Tool-Spaces (Hàm Remote Stateless)

- Gradio apps tag `reachy-mini-tool` + `mcp` trên HF
- Chạy remote trên HF Space, không download
- Mở rộng LLM của conversation app với capabilities mới
- Cài bằng CLI: `tool-spaces add <owner/space>`

| Khía cạnh | Apps | Tool-Spaces |
|-----------|------|-------------|
| Code chạy | Trên robot (local) | Trên HF Space (remote) |
| Cài đặt | Dashboard / REST API | CLI |
| Phạm vi | Hành vi đầy đủ | Một hàm |
| Trust model | Đầy đủ (chạy local) | Sandboxed (remote) |

### JS Apps (Chạy Trong Browser)

Biến thể thứ ba: JS apps tag `reachy_mini_js_app` chạy hoàn toàn trong browser.
Kết nối robot qua WebRTC signaling — zero install, mở URL là chơi.

Ví dụ: [reachy-dance-duo](https://huggingface.co/spaces/TwinPeaksTownie/reachy-dance-duo)
bởi Carson Maestas (community). Hai robot nhảy sync theo nhạc YouTube. Dùng beat
detection real-time, WebRTC audio routing, và Three.js IK visualization. 85
commits, 30 likes, 3 contributors — xây hoàn toàn bằng JS + HF Spaces.

### Bài Học

Rào cản thấp (HF Space = distribution, WebRTC = connectivity, không cần
SSH/firmware) giúp community đóng góp. Đáng cân nhắc cách Autonomous OS có thể
hạ rào cản cho contributors bên ngoài — qua HF Spaces, web-based skill editor,
hoặc plugin format local đơn giản hơn.

## Profile System

Mỗi Reachy Mini có thể có nhiều "profiles" — profile là cấu hình personality:

```
profiles/<name>/
  instructions.txt    # system prompt cho LLM
  tools.txt           # whitelist tool IDs được bật
```

Chuyển profile thay đổi personality và tools khả dụng mà không cần restart.
Tương tự khái niệm hệ thống persona `SOUL.md`.

## Hỗ Trợ Phát Triển Cho AI Agents

Pollen cung cấp hướng dẫn cho AI coding agents qua:

- **AGENTS.md** — giao thức hành vi cho Claude Code, Cursor, Copilot
- **agents.local.md** — context user theo session (loại robot, preferences)
- **skills/*.md** — 12 file tham khảo domain (motion philosophy, safe torque,
  control loops, app creation, debugging, v.v.)
- **CLAUDE.md** — hướng dẫn riêng cho Claude Code

Tương tự `CLAUDE.md` của chúng ta nhưng với tài liệu tham khảo domain phong
phú hơn.

## Repos Chính

| Repository | Mục đích |
|------------|----------|
| [reachy_mini_conversation_app](https://github.com/pollen-robotics/reachy_mini_conversation_app) | Voice app, tool registry, profiles |
| [reachy_mini](https://github.com/pollen-robotics/reachy_mini) | SDK, docs, AGENTS.md, community skills |
| [reachy-mini-os](https://github.com/pollen-robotics/reachy-mini-os) | OS image build (dựa trên pi-gen) |

## Tài Liệu Tham Khảo

- [Thêm MCP Tools cho Reachy Mini (HF Blog)](https://huggingface.co/blog/adding-mcp-tools-to-reachy-mini)
- [Ra mắt Robot App Store (VentureBeat)](https://venturebeat.com/technology/the-app-store-for-robots-has-arrived-hugging-face-launches-open-source-reachy-mini-app-store-with-200-apps)
- [Tạo và Publish Apps (HF Blog)](https://huggingface.co/blog/pollen-robotics/make-and-publish-your-reachy-mini-apps)
- [Thông Số Phần Cứng Reachy Mini](https://huggingface.co/docs/reachy_mini/platforms/reachy_mini/hardware)
- [Jeff Geerling Review](https://www.jeffgeerling.com/blog/2026/testing-reachy-mini-hugging-face-robot/)