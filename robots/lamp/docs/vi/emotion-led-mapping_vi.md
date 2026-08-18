# Emotion → LED + Animation Mapping

Source: `hal/presets.py` — `EMOTION_PRESETS`

| Emotion | Color (RGB) | Hex | Effect | Speed | Servo Animation |
|---|---|---|---|---|---|
| `curious` | 12, 8, 0 | `#0c0800` vàng ấm | candle | 0.3 | curious |
| `happy` | 12, 9, 1 | `#0c0901` vàng | candle | 0.2 | happy_wiggle |
| `sad` | 16, 8, 8 | `#100808` đỏ thẫm | breathing | 0.4 | sad |
| `thinking` | 6, 12, 4 | `#060c04` xanh chìm | pulse | 0.3 | — (xem ghi chú) |
| `idle` | 12, 8, 1 | `#0c0801` vàng dim | breathing | 0.2 | idle |
| `excited` | 12, 8, 12 | `#0c080c` hồng tím | candle | 0.5 | excited |
| `shy` | 16, 7, 2 | `#100702` hồng | breathing | 0.3 | shy |
| `shock` | 12, 12, 12 | `#0c0c0c` trắng dịu | notification_flash | 1.0 | shock |
| `listening` | 4, 8, 16 | `#040810` xanh dương | breathing | 1.2 | — (xem ghi chú) |
| `laugh` | 12, 8, 1 | `#0c0801` vàng sẫm | candle | 0.2 | laugh |
| `confused` | 16, 9, 3 | `#100903` cam đậm | candle | 0.2 | confused |
| `sleepy` | 0, 0, 0 | `#000000` đen (tắt) | solid | — | sleepy |
| `greeting` | 12, 8, 5 | `#0c0805` vàng nhạt | breathing | 0.3 | greeting \| wake_up |
| `goodbye` | 12, 8, 5 | `#0c0805` vàng nhạt | breathing | 0.5 | goodbye |
| `caring` | 12, 8, 6 | `#0c0806` cam hồng | breathing | 0.4 | nod |
| `acknowledge` | 3, 12, 4 | `#030c04` xanh lá | breathing | 0.5 | acknowledge |
| `stretching` | 12, 12, 2 | `#0c0c02` xanh lá nhạt | breathing | 0.6 | stretching |
| `music_strong` | 8, 12, 8 | `#080c08` xanh lá nhạt | rainbow | 1.0 | music_rock |
| `music_chill` | 16, 9, 0 | `#100900` cam | breathing | 0.3 | music_rock \| music_groove \| music_jazz \| music_waltz |
| `scan` | 5, 12, 3 | `#050c03` xanh nhạt | pulse | 0.3 | scanning |
| `nod` | 12, 8, 1 | `#0c0801` cam đất | breathing | 0.5 | nod |
| `headshake` | 16, 6, 1 | `#100601` amber | breathing | 0.5 | headshake |

## `thinking` và `listening` không có servo

Hai emotion này để `"servo": None` — chỉ LED, đèn đứng yên:

- `listening` chạy đúng lúc user đang nói, tiếng servo cộng rung thân máy lọt thẳng vào mic và làm bẩn STT.
- `thinking` bị hook emotion-ack bắn ở **mỗi** message preprocessed, nên có servo là giật liên tục suốt cuộc hội thoại (cũng chính là lý do LED của nó nằm trong `_BACKGROUND_EMOTIONS` guard ở `hal/app_state.py`).

Hai recording `listening.csv` và `thinking_deep.csv` vẫn giữ trong `hal/recordings/`: `/servo/play` gọi tay được, và Reachy vẫn map chúng (`hal/drivers/motors/reachy_service.py`).

Đường code chịu `servo: None` bình thường — `hal/routes/emotion.py` bỏ qua nhánh play, `POST /emotion` trả `"servo": null`. Khác biệt duy nhất: `thinking` dùng LED-restore mặc định 3.5s thay vì tính theo độ dài recording (`listening` vốn không schedule restore).

### `servo: None` một mình KHÔNG làm đèn đứng yên

Không phát recording mới không có nghĩa là đèn im. Idle loop chạy từ lúc boot và lặp vô hạn (`_continue_playback` trong `hal/drivers/motors/animation_service.py`), mà `idle.csv` không hề nhẹ — mỗi vòng 10s nó quét wrist_roll ~32°, wrist_pitch ~26°, base_pitch ~17°. Tệ hơn: emotion vừa chạy xong sẽ **interpolate ngược về idle** trong vài giây, nên cú vung to nhất rơi đúng lúc user đang nói.

Nên với emotion `servo: None`, route gọi `svc.halt()`: drop recording đang chạy, ghim pose hiện tại, torque vẫn ON. Không cần un-halt tường minh — emotion/`/servo/play` kế tiếp gọi `_begin_motion()` và tự xoá cờ.

Hai chốt chặn kèm theo:

- **Music được miễn**: đang phát nhạc thì groove quan trọng hơn, cue listening không được dừng nhảy.
- **Auto-resume idle sau 10s** (`STILL_IDLE_RESUME_SECONDS` trong `hal/routes/emotion.py`): nếu turn không sinh ra emotion nào (LLM lỗi, im lặng sau partial đầu), body tự trở lại idle thay vì đứng chết ở tư thế dở. Bất kỳ `POST /emotion` nào cũng huỷ timer này. Safety net 8s của `voice_service` chỉ dọn LED, không đụng servo — nên timer này là thứ duy nhất lo phần thân.

Đo trên lamp-0c89: sau `listening`, 5 góc servo đứng nguyên ở T+2s / T+5s / T+8s, tới ~T+13s idle chạy lại; `happy` gửi giữa lúc halt thì phát bình thường.

## Ngân sách độ sáng (peak budget)

Emotion LED là **chỉ báo**, không phải chiếu sáng — dùng chung ngân sách với `STATUS_LED_PRESETS`:

- hue thiên xanh lá (xanh lá / vàng / cyan / trắng) → peak channel **12**
- ít hoặc không có xanh lá (đỏ / tím / cam / xanh dương) → peak channel **16**

Mỗi màu được hạ bằng cách **scale tỉ lệ RGB gốc** xuống tier tương ứng, nên hue của từng emotion giữ y như trước. Hạ sáng phải làm bằng scale, không phải chọn màu mới — hue là thứ agent muốn nói, độ sáng chỉ là nói to hay nhỏ.

Gate `light.max_brightness` (lamp: 120) chỉ scale peak **LÊN** tới trần chứ không làm dim, nên hạ sáng phải làm ngay trong preset. Chỉnh xong phải nhìn bằng mắt trên device thật.

`listening` dùng breathing chứ không phải pulse: nó sáng suốt lúc user nói, mà khoảng tối giữa các nhịp của pulse đọc thành cảnh báo khi kéo dài.

Nếu sau này dùng `blink`: `blink()` map speed 1.0 → **~3 Hz** (`hal/drivers/rgb/effects.py`), đủ nhanh để nhức mắt. Giữ ≤ 0.5 (~1.5 Hz trở xuống).

## LED Restore Behavior

- **User đã set color/effect/scene** → sau emotion, restore về màu/scene của user (kèm re-aim nếu là scene)
- **Đèn tắt hoặc chưa set** → emotion LED ở lại sau khi animation xong
- **`shock`** → restore sau 2.0s (notification_flash tự tắt sau ~1.5s)
- **`idle`** → không schedule restore (là ambient resting state)

## Pulse Behavior

Emotion-driven pulse (thinking / listening / scan) chạy trên **nền đen**: wavefront tím/xanh nổi rõ trên strip đen, agent biểu cảm dễ thấy bất kể user đang set màu gì.

Transient pulse (Buddy busy, các driver overlay khác qua `/led/effect` với `transient: true`) thì **overlay trên màu user**: pixel ngoài wavefront giữ màu user, pixel wavefront alpha-blend từ user → emotion. Mục đích: giữ liên tục màu nền user trong khi overlay nhanh.

Source: `hal/drivers/rgb/effects.py:pulse()`; emotion path ở `hal/app_state.py:_apply_emotion_led_display()` (base đen mặc định), transient path ở `hal/routes/led.py:start_led_effect()` (base = `_get_user_base_color()` khi `transient=true`).
