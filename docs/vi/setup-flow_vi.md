# Setup Flow — Tài Liệu

## Tổng Quan

Khi OS server chưa được cấu hình (`SetUpCompleted = false`), thiết bị chạy ở chế độ AP mode, phục vụ Web UI để người dùng setup.

## Flow

```
1. Thiết bị khởi động → check config.json
2. Chưa setup → AP mode (WiFi hotspot)
3. Người dùng kết nối WiFi → mở Web UI
4. Nhập: WiFi SSID/password + LLM config + channel
5. POST /api/device/setup
6. OS Server xử lý (async):
   a. Kết nối WiFi (connect-wifi CLI) — song song, một goroutine
      early-capture publish IP LAN (STA) vào setup state ngay khi wlan0 có
      IP (trước cả khi có internet), để Web UI đọc được lúc AP còn sống
      trong giây lát (xem "Tự Động Chuyển Hướng AP→STA")
   b. Chờ internet (poll 60s)
   c. Lưu config
   d. Ping backend sớm (fire-and-forget HTTP POST {llm_base}/ping, status
      "setting_up") — publish IP LAN mới (local_ip) lên backend mà KHÔNG chờ
      bước setup agent bên dưới, để trang đã mở popup Setup có thể tra IP và
      cứu cú redirect
   e. Setup agent gateway
   f. Chờ agent ready (poll 120s)
   g. SetUpCompleted = true
   h. Ping backend (status "working", setup_completed=true)
7. Nếu thất bại → quay lại AP mode
8. Web UI tự chuyển hướng browser sang http://<lan_ip>/setup ngay khi
   operator đã về Wi-Fi nhà (IP-first; mDNS .local là fallback discovery
   cuối cùng khi AP chết trước lúc đọc được lan_ip)
```

## API

### POST /api/device/setup

```json
{
  "network_ssid": "MyWiFi",
  "network_password": "...",
  "llm_provider": "anthropic",
  "llm_api_key": "sk-...",
  "llm_base_url": "https://api.anthropic.com",
  "llm_model": "claude-haiku-4-5-20251001",
  "channel_type": "telegram",
  "channel_token": "...",
  "channel_id": "...",
  "mqtt_endpoint": "broker.example.com",
  "mqtt_port": 8883,
  "mqtt_username": "...",
  "mqtt_password": "...",
  "fa_channel": "fa/device123",
  "fd_channel": "fd/device123",
  "deepgram_api_key": "..."
}
```

**Response:** Trả về ngay `{"status": 1}`. Setup chạy async trong goroutine sau 2s delay.

**Admin password mặc định:** `admin_password` là optional. Khi để trống ở first-time setup (`SetUpCompleted=false` và chưa có `AdminPasswordHash`), handler mặc định lấy suffix 4 ký tự từ `device.GetDeviceMac()` — cùng suffix mà `scripts/provision/setup-ap.sh` dùng cho AP SSID (`<DEVICE_TYPE>-<xxxx>`). Suffix in trên nhãn dán dưới đế thiết bị, nên user có thể sign in trang admin mà không cần tự đặt password lúc setup. Web UI Setup V2 ẩn hẳn field DEVICE PASSWORD và dựa vào default này; V1 (Device step riêng) vẫn bắt user chọn. Fail 400 (`device hardware ID unreadable`) khi `GetDeviceMac()` trả empty (không có env `DEVICE_TYPE`, không có serial, không có eth MAC) — silent fallback sẽ khiến mọi device không identify được đều có cùng một password well-known.

### POST /api/device/channel

Thay đổi messaging channel sau khi đã setup. Chấp nhận `telegram`, `slack`, `discord`.

**WhatsApp bị reject ở đây** (`400 whatsapp pairing not supported via HTTP; use MQTT add_channel`) — WhatsApp pairing stream rotating QR về caller, HTTP fire-and-forget không carry được. Đường chính tắc là MQTT `add_channel` command (xem `docs/mqtt.md`) — thiết bị publish một message fd_channel cho mỗi pairing event. Re-pair không re-bootstrap dùng MQTT `whatsapp_pair` command.

## Network Setup

1. Gọi `connect-wifi` CLI tool với SSID + password
2. Poll kiểm tra:
   - SSID match? (`iwgetid`)
   - Internet OK? (`ping`)
3. Timeout 60s → fail
4. Thành công → lưu SSID + password vào config

## AP Mode

- Khi chưa setup hoặc setup fail → tự động chuyển AP mode
- Thiết bị phát WiFi hotspot
- Web UI phục vụ trang setup
- `SwitchToAPMode()` trong `system/network/service.go`
- **Tín hiệu LED:** ngay khi HTTP server bắt đầu listen, nếu `SetUpCompleted == false` thì OS server spawn goroutine background (`waitAndPaintSetupReady` trong `server/server.go`) poll `GET /health` của HAL mỗi giây tối đa 30s. Khi `health.led == true` thì fire `POST /led/solid` với `{"color":[255,255,255]}` paint strip trắng solid. Poll vì os-server bind :5000 thường nhanh hơn HAL FastAPI bind :5001 trên cold boot (Python load `rpi_ws281x`, SPI, audio, camera) — fire-and-forget paint sẽ rớt im lặng với `connection refused`. Trắng giữ đến khi setup xong (agent flash + ambient paint đè lên). Blue-breathing booting vẫn show trong lúc init.
- **Khử nhiễu LED trong AP mode:** openclaw WS reconnect loop (`runtimes/openclaw/service_ws.go`) skip Set/Clear `StateAgentDown` khi `config.SetUpCompleted == false`, để overlay cyan disconnect không đè lên trắng setup-needed lúc provisioning. WS vẫn chạy (`device.Setup` cần nó ready để `WaitForAgentReady` pass trước khi flip `SetUpCompleted=true`), chỉ gate side-effect LED thôi.
- **Tín hiệu LED khi join Wi-Fi (`StateWifiConnecting`):** ngay khi setup handler vào `device.Setup()`, kích hoạt `statusled.StateWifiConnecting` (HAL preset `wifi_connecting` = blink màu xanh dương `[0,135,255]` speed 0.5) để ring chuyển từ trắng setup sang blink xanh trong lúc `wlan0` associate. `defer` trong `Setup()` clear state ở mọi return path, nên fail rồi rớt xuống `SwitchToAPMode()` cũng không để strip kẹt blink. Priority nằm trên `Booting` và dưới `OTA`/`Error`/`Connectivity` — tín hiệu này thắng state boot còn sót lại nhưng không che khuất fault thật. Device không có capability `light` sẽ short-circuit trong statusled (no-op).

## Tự Động Chuyển Hướng AP→STA (màn hình "joining Wi-Fi…")

Sau khi operator submit, trang Setup hiện màn hình "Your device is joining
Wi-Fi…" và cố **tự động chuyển hướng browser sang địa chỉ mới của thiết bị trên
Wi-Fi nhà** ngay khi reachable, để operator không phải tự đi tìm IP. Phần này
mô tả cơ chế chuyển hướng đó, tại sao trước đây nó bị kẹt, và đã thay đổi gì.

### Ràng buộc cốt lõi

Thiết bị có **một sóng Wi-Fi duy nhất (`wlan0`)**. Lúc provisioning nó chạy như
access point tại `192.168.100.1`. Để join mạng nhà, nó phải chuyển cùng sóng đó
AP→STA — việc này **tắt AP**. Ngay khi AP chết, browser (vẫn đang nối SSID của
AP) **mất toàn bộ kết nối mạng tới thiết bị** cho tới khi operator tự nối lại
Wi-Fi nhà. **Không có khoảng nào** browser ở trên cả 2 mạng cùng lúc.

Hệ quả: browser chỉ có thể biết IP LAN mới của thiết bị **trước** khi AP tắt,
hoặc qua mDNS **sau** khi operator đã về Wi-Fi nhà.

### Các kênh chuyển hướng (`useSetupStatusPolling.ts`)

Chuyển hướng **ưu tiên IP (IP-first), theo thiết kế.** Tên mDNS `.local` của
thiết bị không phải đích redirect chính: nhiều router gia đình/văn phòng chặn
mDNS multicast (và Android Chrome không có mDNS gốc), nên `.local` im lặng
không resolve được. IP LAN thô resolve trên mọi mạng, nên là nguồn chân lý
ưu tiên cho "thiết bị giờ ở đâu." mDNS chỉ tồn tại như **fallback discovery
cuối cùng** cho trường hợp kênh IP chắc chắn không bao giờ có IP (xem kênh 3).

1. **Phase poll** — poll `GET /api/device/setup/status` qua AP IP khi AP còn
   sống. Đọc `phase` + `lan_ip`. Chết ngay khi AP tắt. (Backend capture IP STA
   sớm để poll này trả được `lan_ip` trong khoảng AP còn sống ngắn ngủi — xem
   `system/device/setup.go`.) Một watchdog theo wall-clock bật cờ `apLost`
   khi poll không được trả lời >5s trong lúc setup đang chạy — dùng wall-clock
   chứ không đếm số lần fail liên tiếp, vì fetch tới AP đã biến mất có thể
   treo nhiều giây trong vòng TCP retry của browser.
2. **LAN-IP probe** — khi đã biết `lan_ip`, probe `http://<lan_ip>/api/health`
   từ browser; khi thành công (operator đã về Wi-Fi nhà và thiết bị online) thì
   chuyển hướng sang `http://<lan_ip>/setup?<params>`. Đây là kênh redirect
   **chính** và luôn giữ quyền quyết định khi nó có đích.
3. **mDNS `.local` fallback probe (chỉ để discovery)** — cứu race của lần setup
   đầu: khi join lần đầu, AP thường tắt **trước khi** wlan0 có DHCP lease, nên
   phase poll không bao giờ đọc được `lan_ip`, máy của operator tự nhảy về
   Wi-Fi nhà/văn phòng, và kênh 2 không có đích — trước đây trang bị kẹt vĩnh
   viễn trên AP IP đã chết. Khi `setupWorking && apLost && !lan_ip` và đã biết
   hostname (`<type>-<xxxx>`, từ trường `mac` của endpoint setup-status mở),
   FE probe `http://<host>.local/api/health` mỗi 2s; thành công thì chuyển
   hướng sang `http://<host>.local/setup?<params>`. Gate theo `apLost` vì lúc
   AP còn sống, avahi có thể trả lời mDNS **qua chính link AP** và một cú
   redirect sớm sẽ reload trang giữa lúc join, mất trạng thái "Setting up…".
4. **`.local` landing seed** — khi trang được phục vụ từ host `.local` (tức
   ngay sau khi kênh 3 bắn), nó fetch `lan_ip` một lần từ endpoint setup-status
   mở, để kênh 2 canonical-upgrade URL về IP thô. `.local` chỉ là cầu
   discovery; IP thô mới là nhà bền vững, vì mDNS có thể ngừng resolve bất kỳ
   lúc nào. Một guard từ chối IP AP `192.168.100.1` để không bao giờ
   "upgrade" nhầm về địa chỉ AP.

**Pre-submit canonical-URL upgrade.** LAN-IP probe ở trên cũng chạy *trước*
submit: khi trang đang ở AP IP (`http://192.168.100.1`) và đã biết `lan_ip`, nó
bật browser khỏi AP IP sắp chết sang `http://<lan_ip>/setup`, địa chỉ sống sót
qua AP→STA. **Trước submit, lúc wlan0 vẫn phục vụ AP và chưa có IP STA, `lan_ip`
rỗng nên trang đơn giản ở lại `192.168.100.1`** — mDNS fallback (kênh 3) gate
theo `setupWorking` nên cũng không bao giờ bắn trước submit.

### Nguyên nhân gốc của bug "kẹt mãi mãi"

Hai lỗi độc lập khiến màn hình treo vô hạn dù thiết bị đã join Wi-Fi thành công:

1. **CSP chặn mọi probe cross-origin.** nginx của thiết bị trả
   `Content-Security-Policy: …; connect-src 'self' ws: wss:`. Trang Setup phục
   vụ từ AP IP, nên `'self'` là `http://192.168.100.1`. Cả probe `lan_ip`
   (`http://172.x.x.x/…`) lẫn probe mDNS (`http://…​.local/…`) đều là **origin
   khác**, nên browser từ chối `fetch` với *"Refused to connect because it
   violates the document's Content Security Policy"* — `mode: "no-cors"`
   **không** bỏ qua được CSP. Mọi probe chết trước khi rời browser.

2. **Kênh LAN-IP gần như không bao giờ có `lan_ip` để dùng.** `lan_ip` chỉ được
   publish vào setup state *sau khi* `SetupNetwork()` xong vòng chờ internet
   **tới 60s**. Nhưng AP tắt trong ~2s sau khi gọi hàm đó, nên phase poll chết
   từ lâu trước khi `lan_ip` tồn tại → kênh LAN-IP bị vô hiệu → chỉ còn kênh
   mDNS → mà trên mạng chặn mDNS thì kênh này cũng không resolve được. Kết quả:
   **không kênh nào chạy.**

Nên trên router chặn mDNS multicast (đúng case thực tế), trang kẹt vĩnh viễn ở
"joining Wi-Fi…" dù đã join hoàn toàn thành công.

### Cách sửa

| Tầng | Thay đổi | Tại sao |
|------|----------|---------|
| **CSP** (`scripts/imager/build*.sh`, `scripts/provision/setup.sh`, `scripts/maintenance/patch-security.sh`) | `connect-src 'self' ws: wss:` → `connect-src 'self' ws: wss: http:` | Cho browser `fetch` probe cross-origin LAN-IP. Phải dùng `http:` (không phải `http://*.local`) vì **CSP không biểu diễn được dải IP** — một token `http:` là cách duy nhất cho phép `http://<bất-kỳ-ip-lan>/…`, nên fix độc lập với subnet của khách (`172.x`, `192.168.x`, `10.x`). |
| **Backend** (`system/device/setup.go`) | Một goroutine poll `GetCurrentIP()` mỗi giây **song song với** `SetupNetwork()` và publish IP STA vào setup state ngay khi xuất hiện (bỏ qua IP AP `192.168.100.1`), trước khi vòng chờ internet 60s xong. | Cho FE **cửa sổ lớn nhất có thể** để đọc `lan_ip` trong khoảng overlap ngắn lúc nó còn poll AP — để kênh LAN-IP thực sự có IP mà chuyển hướng tới. Một guard giữ IP đã capture khỏi bị ghi đè thành chuỗi rỗng bởi lần đọc sau lúc AP đang teardown. |
| **Frontend** (`useSetupStatusPolling.ts`) | Bỏ kênh redirect mDNS `.local` khỏi vai trò *chính*. Kênh redirect chính là LAN-IP probe, carry `pathname + search` và nhắm tới `http://<lan_ip>/setup?<params>`; nó cũng đóng vai trò pre-submit canonical-URL upgrade. (Sau này một fallback `.local` chỉ-để-discovery được thêm lại cho race AP-chết-trước-lan_ip — xem kênh 3–4 ở trên.) | `.local` không đáng tin trên mạng chặn mDNS nên không thể làm đích redirect chính. IP đọc động từ backend — **không hardcode subnet, happy path không phụ thuộc mDNS**. |
| **Frontend** (`Setup.tsx`) | Ô copy "save this address" và link "Continue setup" giờ dùng **URL IP thô** (`http://<lan_ip>/setup`); cả hai màn gating theo `setupLanIP` thay vì mDNS host, fallback về gợi ý router-admin khi chưa biết IP. | IP-only từ đầu đến cuối — operator không bao giờ bị đưa địa chỉ `.local` không resolve được trên mạng của họ. |
| **Frontend** (`Setup.tsx`) | Nút Copy thêm fallback `document.execCommand("copy")` (textarea ẩn) cho khi `navigator.clipboard` không có. | Trang Setup phục vụ qua HTTP thuần (`http://192.168.100.1`), nơi `navigator.clipboard` là `undefined` (cần secure context) — nên API mới im lặng không làm gì và nút không hoạt động. Đường legacy chạy được trên origin `http://`. |

### Đích chuyển hướng

Happy path giờ chuyển hướng tới **`http://<lan_ip>/setup?<params>`** (vd
`http://172.168.20.145/setup?…`) — IP thô, hoạt động bất kể mDNS. Tới khi
early-poll lấy được `lan_ip`, link copy thủ công fallback về
`http://<type>-<id>.local/setup?<params>`.

### Đánh giá & đánh đổi

- **Sửa được gì:** auto-redirect (và link copy thủ công) giờ hoạt động trên
  mạng chặn mDNS — đúng lỗi thực tế đã báo. Giải pháp **không phụ thuộc subnet**
  — không giả định dải IP private cụ thể nào.
- **Vẫn phụ thuộc gì:** auto-redirect qua kênh LAN-IP chỉ chạy nếu FE kịp
  capture `lan_ip` trong khoảng ~2s lúc AP còn sống — với lần setup đầu, cửa
  sổ này thường đóng trước khi DHCP xong. mDNS fallback (kênh 3) cover case
  đó trên các mạng resolve được `.local`; trên mạng chặn mDNS mà không capture
  được `lan_ip` thì không kênh tự động nào bắn được, và **nhập IP thủ công là
  fallback chắc chắn** — operator tra IP thiết bị trong router rồi gõ vào, nên
  không bao giờ bị kẹt.
- **Backend rendezvous (phía device đã sẵn sàng):** cú ping backend sớm (bước
  6d) publish `local_ip` ngay khi WiFi lên, nên trang đã mở popup Setup (vd
  autonomous.ai) có thể poll backend theo `mac` rồi navigate popup sang
  `http://<ip>/setup?<params>` — opener được phép *navigate* popup
  cross-origin dù không đọc được. Cách này tự động cover cả mạng chặn mDNS,
  nhưng cần backend lưu/expose IP và parent page chịu poll; cả hai đều nằm
  ngoài repo này.
- **Đánh đổi bảo mật của `http:` trong CSP:** `connect-src http:` cho phép trang
  Setup `fetch` mọi origin HTTP thuần, không chỉ thiết bị. Chấp nhận được vì
  bundle Setup chỉ phục vụ trên LAN/AP, không gửi secret trong các health probe
  này, và CSP không có cách hẹp hơn để cho phép một IP LAN tùy ý. Ghi nhận tại
  `docs/security/CHECKLIST.md` (F9).
- **Tại sao không tránh hẳn việc tắt AP:** dùng 2 sóng hoặc đường có dây sẽ bỏ
  được ràng buộc, nhưng phần cứng đích chỉ có một sóng Wi-Fi — nên mô hình "biết
  IP trước khi AP chết, hoặc qua mDNS sau đó" là cố hữu.

### Join thất bại (sai mật khẩu Wi-Fi)

Sai mật khẩu Wi-Fi là lỗi setup phổ biến nhất, và nó phơi bày đúng ràng buộc
một-radio ở chiều ngược lại: operator không bao giờ biết được lý do.

**Vì sao kết luận của backend không ai nghe được.** `SetupNetwork` poll tối đa
60s trước khi trả lỗi, nhưng AP đã tear down chỉ ~2s sau khi submit. Từ thời
điểm đó phase poll không còn với tới `192.168.100.1` được nữa, và máy của
operator thường đã tự nhảy về Wi-Fi nhà. Backend set `phase="failed"` ở t≈62s và
`handler.Setup` gọi `SwitchToAPMode()`, khôi phục hotspot sau ~5-8s — nhưng lúc
đó phía browser không còn ai lắng nghe. Trước khi có fix bên dưới, trang quay
"joining Wi-Fi…" vĩnh viễn.

Ba cơ chế hiện xử lý việc này:

1. **Timeout phía client** (`useSetupStatusPolling.ts`, `JOIN_TIMEOUT_SEC = 80`).
   Khi join vẫn đang `connecting`, AP đã mất liên lạc (`apLost`), và chưa từng
   bắt được `lan_ip`, FE tự kết luận thất bại. Mốc 80s nằm sau timeout 60s của
   backend cộng thời gian khôi phục AP, nên một lần join chỉ đơn thuần chậm sẽ
   không bao giờ bị kết luận nhầm. Message được diễn đạt dè dặt ("không kết nối
   được… thường do sai mật khẩu Wi-Fi, hoặc mạng chỉ có 5GHz") vì lý do thật
   chưa bao giờ tới nơi — khẳng định một nguyên nhân cụ thể sẽ là đoán mò.

2. **Nhận trạng thái failed lúc mount** (`useSetupController.ts`). `setupState`
   nằm trong RAM của os-server và sống sót qua chu trình AP→STA→AP, còn
   `GET /api/device/setup/status` là endpoint public (thiết bị setup thất bại
   chưa từng ghi admin hash, nên endpoint có admin-gate sẽ không đọc được).
   Controller đọc nó một lần lúc mount; nếu lần thử trước `failed`, nó vào thẳng
   màn hình lỗi kèm message **thật** từ backend. Đây là thứ cứu được trường hợp
   popup mở lại, tab bị đóng, hoặc user tự reload — trước đây tất cả đều mount ra
   form Wi-Fi trống trơn, không hề báo là đã có gì thất bại. Chỉ nhận `failed`;
   một `connecting` cũ sẽ chiếm quyền tab và đẩy nó vào màn hình progress mà nó
   không thể điều khiển.

   **Việc nhận trạng thái này cũng clear luôn state của lần thử thất bại.** Nối
   lại AP của thiết bị sau khi lỗi sẽ cho operator một wizard thật sự sạch:
   `ssid`, `password`, `adminPassword`, `error`, `stepError` và `setupLanIP` đều
   được reset, và wizard quay về bước Wi-Fi. Lỗi Wi-Fi bail trước khi
   `device.Setup` ghi bất kỳ config nào, nên không có gì trong số đó từng được
   lưu phía thiết bị — giữ lại chỉ khiến operator submit lại đúng thông tin vừa
   sai mà không nhận ra. Snapshot param đã lưu cũng bị xoá
   (`clearStoredSetupParams()`): `sessionStorage` là per-tab nhưng **sống sót
   qua F5**, nên operator để nguyên tab, nối lại AP rồi reload sẽ rehydrate lại
   query string của lần thất bại vào một form trông sạch nhưng vẫn ngậm
   `llm_api_key` cũ. Xoá ngay lúc mount (thay vì reload luôn) giữ cho document
   hiện tại vẫn dùng được cho retry tại chỗ "Back to Wi-Fi", đồng thời đảm bảo
   lần load **kế tiếp** bắt đầu từ con số không.

   Trạng thái failed được nhận sẽ set cờ riêng `adoptedFailure` thay vì dùng lại
   `setupWorking`. Hai thứ này mang ý nghĩa khác nhau: `setupWorking` nghĩa là
   *tab này đang có một lần join chạy dở*, và nó khởi động phase poll 600ms, bộ
   đếm elapsed, cùng các emit connecting/failed của bridge. Bật nó cho một lần
   join mà ta chỉ đọc được sẽ khởi động lại poll trong khi backend vẫn báo
   `phase="failed"` — không có gì reset `setupState` về `idle` — nên poll sẽ
   khẳng định lại trạng thái lỗi mỗi 600ms và đá operator ra khỏi form Wi-Fi mỗi
   lần họ bấm "Back to Wi-Fi". Màn hình render theo
   `showProgressScreen = setupWorking || adoptedFailure`; hành động retry clear
   cả hai.

   **Chốt chặn verdict cũ trong poll.** `handler.Setup` trả `200` ngay nhưng
   hoãn `device.Setup` 2s, và không có gì đưa `setupState` về `idle` trong
   khoảng đó — nên trong ~2s sau khi submit lại, backend vẫn báo `phase="failed"`
   của lần **trước**. Vì vậy poll bỏ qua mọi verdict cuối cho tới khi thấy
   `phase="connecting"` xác nhận lần chạy hiện tại đã bắt đầu; không có chốt
   này, lần poll đầu tiên sau khi retry sẽ ném operator thẳng về màn hình lỗi.

3. **Retry thật sự validate được.** `mergeMissingFromConfig` không còn bị gate
   bởi `SetUpCompleted`. Lỗi Wi-Fi bail trước khi `device.Setup` ghi bất kỳ
   config nào, nên thiết bị vẫn ở `SetUpCompleted=false` — trong khi browser có
   thể đã mất credential được đẩy vào (sessionStorage chết theo tab; popup mở lại
   không kèm query string gốc thì rỗng). Cái gate đó khiến retry fail validation
   ở `LLMAPIKey` và hiện *"Missing: AI Brain API key"* cho người chỉ gõ sai mật
   khẩu Wi-Fi. Merge chỉ điền vào các ô đang rỗng, lấy từ chính config của thiết
   bị, nên không thể ghi đè thứ operator gửi lên, cũng không lộ thêm gì mới.

**UI khôi phục.** Màn hình lỗi nêu rõ tên hotspot cần nối lại
(`<type>-<suffix>`, cùng chuỗi với mDNS host — `setup-ap.sh` suy ra cả hai từ
hardware ID), vì join thất bại để lại operator trên mạng nhà và không nút nào
với tới thiết bị được cho đến khi họ chuyển mạng lại. Hai hành động:

- **Back to Wi-Fi** — retry tại chỗ. Xoá mật khẩu vừa sai (để không bị gửi lại y
  nguyên) nhưng giữ SSID và giữ nguyên document hiện tại, gồm cả các param nó
  được mở kèm. Đây là mặc định hợp lý cho lỗi gõ nhầm.
- **Start over** — reset cứng (`resetSetupSession()`): xoá snapshot param trong
  sessionStorage và **hard-reload** về `/setup` trơn, tạo ra đúng trạng thái của
  lần đầu tiên vào AP. Việc reload là **bắt buộc**, không phải cho đẹp:
  `useSetupUrlParams` snapshot `INITIAL_SEARCH` ở *module load*, nên chỉ reset
  React state sẽ để lại `llm_api_key` / `device_id` cũ còn sống trong module đó
  và form "sạch" vẫn submit chúng. Bắn `start_over_clicked` trước, để parent nào
  muốn giữ config đã đẩy có thể mở lại popup bằng URL mới. Tuỳ chọn theme được
  giữ lại có chủ ý — đó là lựa chọn của người dùng, không liên quan tới lần thử
  này.

**Theo từng device: bỏ qua màn hình lỗi.** `intern-v2` hoàn toàn không hiện màn
hình lỗi. Join thất bại sẽ đưa operator thẳng về form Wi-Fi — đúng trạng thái
cuối như khi bấm "Back to Wi-Fi" — không banner lỗi, không checklist, không gợi
ý nối lại hotspot. Đây là quyết định sản phẩm cho riêng device class đó; `lamp`,
`reachy-mini` và `unitree-go2w` giữ nguyên màn hình đầy đủ.

Device class lấy từ `mac` (`"<device_type>-<4 hex>"`, ví dụ `intern-v2-d94b`),
bỏ đi phần hex đuôi — **không** lấy từ URL param, vì browser của operator không
được phép tác động vào giá trị này (xem `DeviceTypeOrDefault`, nơi coi
`DEVICE_TYPE` là hardware identity bất biến). Cả hai đường dẫn tới trạng thái
lỗi đều được xử lý:

- **Timeout trong tab hiện tại** — `showProgressScreen` chặn màn hình, và một
  effect gọi đúng `retryFromFailure()` mà nút vẫn dùng, nên hai đường không thể
  lệch nhau.
- **Nhận trạng thái lúc mount** — effect mount đọc device class từ **chính**
  response `setup/status` mang verdict đó, thay vì chờ state `mac` (được điền
  bởi một request khác) — nếu chờ thì màn hình lỗi sẽ kịp hiện một frame rồi mới
  bị chặn. Nó vẫn clear state của lần thử như thường lệ nhưng không bao giờ bật
  cờ failure.

Cả hai đường vẫn bắn `setup_failed` qua bridge. Màn hình chỉ bị ẩn với
**operator**, không ẩn với companion app — đường adoption bắn emit trực tiếp, vì
effect thường gửi nó bị gate bởi `setupWorking`, thứ mà đường này cố tình không
bật.

**Chưa xử lý.** Lý do từ phía thiết bị vẫn còn thô: `SetupNetwork` chỉ poll
`CheckInternet()` + so khớp SSID, nên sai mật khẩu, mạng chỉ có 5GHz, và router
từ chối client đều hiện ra như nhau là
`"no internet or SSID did not match within 60s"` sau trọn 60s.
`wpa_supplicant` biết ngay sự khác nhau (`4WAY_HANDSHAKE_FAILED`, `WRONG_KEY`);
đọc `wpa_cli status` trong vòng poll sẽ fail trong ~5s với nguyên nhân chính xác
— lúc đó AP vẫn còn sống nên phase poll sẽ chuyển được nó về, và timeout ở trên
sẽ trở thành fallback hiếm dùng thay vì con đường chính.

### Đánh dấu bước Wi-Fi đã xong sau khi reload

Auto-redirect đưa trang vào một **full page reload trên origin LAN-IP mới**
(`http://<lan_ip>/setup?…`), là một origin khác với trang AP. Toàn bộ React
state của form Setup — kể cả `ssid` / `password` operator vừa nhập — reset về
rỗng. `/api/device/config`, thứ lẽ ra rehydrate lại SSID đã lưu, bị admin-gate và
thiết bị mới chưa có admin session (401), nên trang sau reload không đọc lại
được. Nếu để nguyên, bước Wi-Fi sẽ render lại "Choose your Wi-Fi + nhập password"
dù thiết bị đã ở trên Wi-Fi nhà.

Để tránh hỏi lại, trạng thái done của bước Wi-Fi được suy ra từ **trạng thái
mạng sống của thiết bị**, không phải từ các ô form (`useWifiConnected.ts`):

- `GET /api/network/check-internet` — thiết bị có uplink. AP setup không có
  uplink, nên có internet == thiết bị đã rời AP mode và join một mạng thật.
- `GET /api/network/current` — SSID mà `wlan0` đang associated (`iwgetid -r`);
  non-empty == đang ở station mode.

Cả hai đều **public** (không cần admin auth), khớp đúng tín hiệu internet mà
`SetupGate` (`App.tsx`) đã dùng để chọn continue vs initial mode. Khi cả hai
thỏa, `sectionDone.wifi` short-circuit thành done và `WifiSection` thu gọn
picker thành một hàng **"Connected to `<ssid>`"** (kèm link *Change network* để
đổi mạng) thay cho selector rỗng. SSID đang associated cũng được prefill vào
picker. *Password* Wi-Fi không bao giờ rời thiết bị — chỉ tên SSID đang associated
(thứ thiết bị vốn đã scan và broadcast) và boolean `check-internet` được đọc.

Trong lúc probe đầu tiên còn đang chạy (`checking`), `WifiSection` render một
**skeleton** cho ô network + password thay vì picker rỗng, để bước này không
flash "Choose your Wi-Fi" một nhịp trước khi resolve sang trạng thái connected.
Các retry nền về sau (xử lý race DHCP-lease) không raise lại skeleton, nên picker
giữ nguyên tương tác được sau khi đã hiện.

### Deep-link vào một bước qua URL hash

Redirect carry nguyên hash, nên một URL kiểu `http://<lan_ip>/setup?<params>#voice`
phải mở thẳng tab **Voice**. Lúc mount, `Setup.tsx` đọc `window.location.hash` và
khi nó trỏ tới một section đang hiển thị (`#wifi` / `#voice` / `#face` / …) thì
chọn bước đó. Vì Voice/Face chỉ tồn tại ở continue mode, việc này dựa vào
`SetupGate` resolve xong mode *trước khi* Setup mount (nó render `null` cho tới
lúc đó).

Auto-scroll của continue mode — vốn nhảy operator tới bước *chưa hoàn thành* đầu
tiên — bị chặn khi một deep-link hash hợp lệ đã được honor (`deepLinkedRef`), nên
`#voice` không bị ghi đè ngược về Wi-Fi ngay. Hành vi "tất cả bước bắt buộc xong
→ bounce về `/monitor`" vẫn được giữ.

## Post-Setup

Sau khi `SetUpCompleted = true`:
1. Kết nối OpenClaw WebSocket
2. Kết nối MQTT (subscribe fa_channel)
3. Start voice pipeline (nếu có Deepgram key)
4. Start ambient idle behaviors
5. Start sensing loop

## Config

Config lưu tại `config/config.json`. Managed bởi `server/config/config.go`.

| Field | Mô tả |
|-------|-------|
| `SetUpCompleted` | `true` khi setup xong |
| `NetworkSSID` | WiFi SSID |
| `NetworkPassword` | WiFi password |
| `LLMProvider` | anthropic, openai, google, ... |
| `LLMApiKey` | API key cho LLM |
| `LLMBaseUrl` | LLM API base URL |
| `LLMModel` | Model name |
| `ChannelType` | telegram, slack |
| `ChannelToken` | Channel bot token |
| `ChannelID` | Channel/chat ID |
| `DeepgramApiKey` | Deepgram STT API key |
| `LocalIntent` | Enable/disable local intent matching (default: true) |
| `MQTTEndpoint` | MQTT broker host |
| `MQTTPort` | MQTT broker port |
| `FAChannel` | MQTT subscribe topic (server→device) |
| `FDChannel` | MQTT publish topic (device→server) |

## Bridge sự kiện về cửa sổ cha (parent window)

Khi trang Setup được mở dạng popup/iframe từ một site khác (ví dụ
`autonomous.ai`), nó báo từng cột mốc ngược về cửa sổ đã mở nó qua
`window.postMessage`. Đây là kênh cross-origin duy nhất hoạt động được
popup→opener, vì Setup được phục vụ từ AP IP của thiết bị
(`http://192.168.100.1`) hoặc host `<type>-<id>.local` — khác origin.

Cửa sổ cha nên truyền origin của mình để thiết bị biết gửi về đâu và payload
không bị broadcast ra `*`:

```js
const origin = encodeURIComponent(window.location.origin);
window.open(`http://192.168.100.1/setup?parent_origin=${origin}&...`, "_blank");
```

Thứ tự resolve origin: `?parent_origin=` → origin của `document.referrer` → `*`.

Mỗi message là một JSON envelope phẳng:
`{ source: "autonomous-device-setup", v: 1, event, ts, ...data }`. Lọc theo
`source` và switch theo `event`:

| `event` | Khi nào | Trường thêm |
|---------|---------|-------------|
| `setup_opened` | Wizard đã mount | `mode`, `deviceId`, `mac` |
| `step_changed` | Operator đổi bước wizard | `step` |
| `wifi_selected` | Đã chọn một mạng WiFi | `ssid` |
| `setup_submitted` | Bấm "Setup", chuẩn bị gửi request | `ssid`, `channel` |
| `setup_error` | Có lỗi validation/backend | `message` |
| `setup_connecting` | Thiết bị đang join WiFi (sau submit) | — |
| `setup_connected` | Thiết bị online + reachable | `mdns_host`, `lan_ip` |
| `setup_failed` | Join WiFi thất bại | `message` |
| `retry_clicked` | Bấm "Back to Wi-Fi" sau khi lỗi | — |
| `start_over_clicked` | Bấm "Start over" sau khi lỗi — popup sắp hard-reload và **bỏ toàn bộ param nó được mở kèm**; mở lại bằng URL mới nếu cần giữ config đã đẩy | — |
| `continue_clicked` | Bấm "Continue setup →" | `mdns_host` |
| `monitor_clicked` | Bấm "Go to monitor →" | — |

Các emit đều best-effort: không có opener/parent thì là no-op, và lỗi
postMessage bị nuốt, nên bridge không bao giờ ảnh hưởng tới luồng setup. Ví dụ
listener đầy đủ nằm ở phần header của file `lib/setupBridge.ts`.

## Code

| File | Vai trò |
|------|---------|
| `system/device/setup.go` | Setup orchestration + goroutine early-capture IP LAN |
| `system/web/src/lib/setupBridge.ts` | Bridge sự kiện về cửa sổ cha (postMessage) |
| `system/web/src/pages/Setup.tsx` | UI wizard Setup + các điểm gọi emit bridge + link copy ưu tiên IP |
| `system/web/src/hooks/setup/useSetupStatusPolling.ts` | Auto-redirect AP→STA: phase poll + LAN-IP probe + mDNS probe |
| `system/web/src/hooks/setup/useWifiConnected.ts` | Nhận biết Wi-Fi-đã-xong sau reload từ trạng thái sống của thiết bị (`check-internet` + `network/current`) |
| `system/network/service.go` | WiFi connect, AP mode, `CurrentNetwork()` (SSID đang associated) |
| `system/server/device/delivery/http/handler.go` | HTTP setup handler (goroutine async) |
| `system/server/config/config.go` | Config load/save |
| `scripts/imager/build-orangepi.sh`, `scripts/imager/build.sh`, `scripts/provision/setup.sh` | nginx config bake vào image (gồm CSP `connect-src`) |
| `scripts/maintenance/patch-security.sh` | Patch bảo mật OTA cho thiết bị đã provision (migrate CSP) |
