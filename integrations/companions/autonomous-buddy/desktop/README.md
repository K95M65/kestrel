# Kestrel Buddy — Windows, Linux, and Mac (desktop)

Go companion that pairs with the robot over the same protocol as the Mac menu-bar app. Opens a small local page that uses the desk chrome (cream / ocean).

Mac still has the native menu-bar build in `../macos/` for Accessibility click/type. This binary is the path for **Windows** and **Linux**, and a fallback on Mac.

## What it can do

| Action | Windows | Linux | Mac (this binary) |
|---|---|---|---|
| `open_url` | yes | xdg-open | `open` |
| `open_app` | `start` | PATH / gtk-launch | `open -a` |
| `close_app` | taskkill | pkill | AppleScript quit |
| `notification` | toast | notify-send | osascript |
| `write/read_clipboard` | PowerShell | wl-copy / xclip | pbcopy |
| `type_text` | SendKeys | xdotool / ydotool | System Events |
| click / screenshot / key_combo | not yet | not yet | use the Mac app |

## Run

```bash
cd desktop
go test ./...
go build -o autonomous-buddy .
./autonomous-buddy                 # opens http://127.0.0.1:18791
./autonomous-buddy -listen 127.0.0.1:0
```

On the robot: Home or House → Uses → Websites → pair with the 6-digit code.

Cross-compile:

```bash
GOOS=windows GOARCH=amd64 go build -o autonomous-buddy.exe .
GOOS=linux   GOARCH=amd64 go build -o autonomous-buddy-linux .
```
