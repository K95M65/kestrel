# Emotion → LED + Animation Mapping

Source: `hal/presets.py` — `EMOTION_PRESETS`

| Emotion | Color (RGB) | Hex | Effect | Speed | Servo Animation |
|---|---|---|---|---|---|
| `curious` | 25, 17, 0 | `#191100` amber | breathing | 1.0 | curious |
| `happy` | 25, 20, 0 | `#191400` vàng | candle | 1.0 | happy_wiggle |
| `sad` | 5, 5, 18 | `#050512` xanh trầm | breathing | 0.8 | sad |
| `thinking` | 15, 7, 25 | `#0f0719` tím lavender | pulse | 1.5 | thinking_deep |
| `idle` | 16, 22, 22 | `#101616` xanh nhạt | breathing | 0.8 | idle |
| `excited` | 22, 3, 22 | `#160316` hồng tím | blink | 2.5 | excited |
| `shy` | 25, 12, 15 | `#190c0f` hồng | blink | 0.5 | shy |
| `shock` | 25, 25, 25 | `#191919` trắng dịu | notification_flash | 2.0 | shock |
| `listening` | 3, 9, 22 | `#030916` xanh dương | pulse | 1.5 | listening |
| `laugh` | 22, 17, 3 | `#161103` vàng sẫm | blink | 1.2 | laugh |
| `confused` | 21, 4, 1 | `#150401` cam đậm | candle | 0.6 | confused |
| `sleepy` | 3, 2, 9 | `#030209` chàm rất tối | breathing | 0.5 | sleepy |
| `greeting` | 25, 15, 7 | `#190f07` vàng nhạt | blink | 0.8 | greeting | wake_up |
| `goodbye` | 25, 15, 7 | `#190f07` vàng nhạt | breathing | 0.5 | goodbye |
| `caring` | 25, 13, 9 | `#190d09` cam hồng | breathing | 0.4 | nod |
| `acknowledge` | 3, 22, 11 | `#03160b` xanh lá | blink | 1.0 | acknowledge |
| `stretching` | 24, 23, 22 | `#181716` trắng ngà | breathing | 0.6 | stretching |
| `music_strong` | 13, 20, 13 | `#0d140d` xanh lá nhạt | rainbow | 1.5 | music_rock |
| `music_chill` | 25, 10, 0 | `#190a00` cam | breathing | 0.5 | music_rock | music_groove | music_jazz | music_waltz |
| `scan` | 2, 16, 21 | `#021015` cyan | pulse | 2.0 | scanning |
| `nod` | 3, 22, 11 | `#03160b` xanh lá | blink | 1.0 | nod |
| `headshake` | 22, 3, 3 | `#160303` đỏ | blink | 1.0 | headshake |

Độ sáng: mọi cue giới hạn peak channel 25 — đây là **chỉ báo**, không phải chiếu sáng. Gate `light.max_brightness` (lamp: 120) chỉ scale peak LÊN tới trần, không làm dim, nên việc hạ sáng phải làm ngay trong preset.

## LED Restore Behavior

- **User đã set color/effect/scene** → sau emotion, restore về màu/scene của user (kèm re-aim nếu là scene)
- **Đèn tắt hoặc chưa set** → emotion LED ở lại sau khi animation xong
- **`shock`** → restore sau 2.0s (notification_flash tự tắt sau ~1.5s)
- **`idle`** → không schedule restore (là ambient resting state)

## Pulse Behavior

Emotion-driven pulse (thinking / listening / scan) chạy trên **nền đen**: wavefront tím/xanh nổi rõ trên strip đen, agent biểu cảm dễ thấy bất kể user đang set màu gì.

Transient pulse (Buddy busy, các driver overlay khác qua `/led/effect` với `transient: true`) thì **overlay trên màu user**: pixel ngoài wavefront giữ màu user, pixel wavefront alpha-blend từ user → emotion. Mục đích: giữ liên tục màu nền user trong khi overlay nhanh.

Source: `hal/drivers/rgb/effects.py:pulse()`; emotion path ở `hal/app_state.py:_apply_emotion_led_display()` (base đen mặc định), transient path ở `hal/routes/led.py:start_led_effect()` (base = `_get_user_base_color()` khi `transient=true`).
