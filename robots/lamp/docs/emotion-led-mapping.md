# Emotion → LED + Animation Mapping

Source: `hal/presets.py` — `EMOTION_PRESETS`

| Emotion | Color (RGB) | Hex | Effect | Speed | Servo Animation |
|---|---|---|---|---|---|
| `curious` | 12, 8, 0 | `#0c0800` amber | breathing | 1.0 | curious |
| `happy` | 12, 10, 0 | `#0c0a00` vàng | candle | 1.0 | happy_wiggle |
| `sad` | 4, 4, 16 | `#040410` xanh trầm | breathing | 0.8 | sad |
| `thinking` | 10, 4, 16 | `#0a0410` tím lavender | pulse | 1.5 | thinking_deep |
| `idle` | 9, 12, 12 | `#090c0c` xanh nhạt | breathing | 0.8 | idle |
| `excited` | 16, 2, 16 | `#100210` hồng tím | blink | 0.5 | excited |
| `shy` | 12, 6, 7 | `#0c0607` hồng | blink | 0.25 | shy |
| `shock` | 12, 12, 12 | `#0c0c0c` trắng dịu | notification_flash | 2.0 | shock |
| `listening` | 2, 7, 16 | `#020710` xanh dương | breathing | 1.2 | listening |
| `laugh` | 12, 9, 2 | `#0c0902` vàng sẫm | blink | 0.4 | laugh |
| `confused` | 16, 3, 1 | `#100301` cam đậm | candle | 0.6 | confused |
| `sleepy` | 3, 2, 9 | `#030209` chàm rất tối | breathing | 0.5 | sleepy |
| `greeting` | 12, 7, 3 | `#0c0703` vàng nhạt | blink | 0.3 | greeting | wake_up |
| `goodbye` | 12, 7, 3 | `#0c0703` vàng nhạt | breathing | 0.5 | goodbye |
| `caring` | 12, 6, 4 | `#0c0604` cam hồng | breathing | 0.4 | nod |
| `acknowledge` | 1, 8, 4 | `#010804` xanh lá | blink | 0.3 | acknowledge |
| `stretching` | 12, 12, 11 | `#0c0c0b` trắng ngà | breathing | 0.6 | stretching |
| `music_strong` | 8, 12, 8 | `#080c08` xanh lá nhạt | rainbow | 1.5 | music_rock |
| `music_chill` | 16, 6, 0 | `#100600` cam | breathing | 0.5 | music_rock | music_groove | music_jazz | music_waltz |
| `scan` | 1, 9, 12 | `#01090c` cyan | pulse | 2.0 | scanning |
| `nod` | 1, 8, 4 | `#010804` xanh lá | blink | 0.3 | nod |
| `headshake` | 16, 2, 2 | `#100202` đỏ | blink | 0.35 | headshake |

## Ngân sách độ sáng (peak budget)

Emotion LED là **chỉ báo**, không phải chiếu sáng — dùng chung ngân sách với `STATUS_LED_PRESETS`:

- hue thiên xanh lá (xanh lá / vàng / cyan / trắng) → peak channel **12**
- ít hoặc không có xanh lá (đỏ / tím / cam / xanh dương) → peak channel **16**
- `sleepy` là ngoại lệ ở peak 9 (cố ý mờ nhất dàn, vẫn trên sàn 8)

Gate `light.max_brightness` (lamp: 120) chỉ scale peak **LÊN** tới trần chứ không làm dim, nên hạ sáng phải làm ngay trong preset. Chỉnh xong phải nhìn bằng mắt trên device thật — số nhìn trên màn hình không phản ánh đúng.

## Tốc độ blink

`blink()` map speed 1.0 → **~3 Hz** (`hal/drivers/rgb/effects.py`), đủ nhanh để gây nhức mắt khi strip nằm ngang tầm mắt. Giữ mọi cue blink ở **≤ 0.5** (~1.5 Hz trở xuống): blink đã đọc ra là "đang chớp" từ rất lâu trước khi đủ nhanh để khó chịu, và cái phân biệt nó với breathing/pulse là **hình dạng**, không phải tần số.

`listening` dùng breathing chứ không phải pulse: nó sáng suốt lúc user nói, mà khoảng tối giữa các nhịp của pulse đọc thành cảnh báo khi kéo dài. Phân biệt với `idle` bằng hue (xanh dương vs xanh nhạt) + nhịp nhanh hơn (1.2 vs 0.8).

## LED Restore Behavior

- **User đã set color/effect/scene** → sau emotion, restore về màu/scene của user (kèm re-aim nếu là scene)
- **Đèn tắt hoặc chưa set** → emotion LED ở lại sau khi animation xong
- **`shock`** → restore sau 2.0s (notification_flash tự tắt sau ~1.5s)
- **`idle`** → không schedule restore (là ambient resting state)

## Pulse Behavior

Emotion-driven pulse (thinking / listening / scan) chạy trên **nền đen**: wavefront tím/xanh nổi rõ trên strip đen, agent biểu cảm dễ thấy bất kể user đang set màu gì.

Transient pulse (Buddy busy, các driver overlay khác qua `/led/effect` với `transient: true`) thì **overlay trên màu user**: pixel ngoài wavefront giữ màu user, pixel wavefront alpha-blend từ user → emotion. Mục đích: giữ liên tục màu nền user trong khi overlay nhanh.

Source: `hal/drivers/rgb/effects.py:pulse()`; emotion path ở `hal/app_state.py:_apply_emotion_led_display()` (base đen mặc định), transient path ở `hal/routes/led.py:start_led_effect()` (base = `_get_user_base_color()` khi `transient=true`).
