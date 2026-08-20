import AppKit

/// Desk chrome (cream / ocean) — same tokens as system/web `index.css` `.lm-root`.
enum Brand {
    static let cream = NSColor(srgbRed: 0xF2 / 255, green: 0xEF / 255, blue: 0xE7 / 255, alpha: 1)
    static let card = NSColor(srgbRed: 1, green: 0xFC / 255, blue: 0xF7 / 255, alpha: 1)
    static let ocean = NSColor(srgbRed: 0x33 / 255, green: 0x68 / 255, blue: 0xA0 / 255, alpha: 1)
    static let ink = NSColor(srgbRed: 0x1E / 255, green: 0x3A / 255, blue: 0x54 / 255, alpha: 1)
    static let muted = NSColor(srgbRed: 0x66 / 255, green: 0xA3 / 255, blue: 0xBF / 255, alpha: 1)
    static let onOcean = cream
}
