#!/usr/bin/env swift
// make_dmg_bg.swift — generates a DMG installer background image using AppKit.
// Usage: swift assets/make_dmg_bg.swift <app-icon.png> <output-1x.png> <output-2x.png>
//
// Produces:
//   output-1x.png  — 660×400 standard resolution
//   output-2x.png  — 1320×800 retina (@2x)
//
// The background shows:
//   • Talkback app icon (left)
//   • Arrow pointing right (centre)
//   • macOS Applications folder icon (right)
//   • "Drag Talkback to Applications to install" caption

import AppKit
import Foundation

// ── helpers ──────────────────────────────────────────────────────────────────

func renderBackground(size: NSSize, appIconPath: String) -> NSBitmapImageRep {
    let W = size.width, H = size.height

    let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil,
        pixelsWide: Int(W), pixelsHigh: Int(H),
        bitsPerSample: 8, samplesPerPixel: 4,
        hasAlpha: true, isPlanar: false,
        colorSpaceName: .deviceRGB,
        bytesPerRow: 0, bitsPerPixel: 0)!

    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)

    // ── background gradient ──────────────────────────────────────────────────
    let gradient = NSGradient(
        colors: [
            NSColor(white: 0.90, alpha: 1.0),
            NSColor(white: 0.97, alpha: 1.0),
        ],
        atLocations: [0.0, 1.0],
        colorSpace: .genericGray)!
    gradient.draw(in: NSRect(x: 0, y: 0, width: W, height: H), angle: 90)

    // ── subtle rounded rect panel ─────────────────────────────────────────────
    let panelRect = NSRect(x: W * 0.07, y: H * 0.18, width: W * 0.86, height: H * 0.65)
    let panelPath = NSBezierPath(roundedRect: panelRect, xRadius: 18, yRadius: 18)
    NSColor(white: 1.0, alpha: 0.55).setFill()
    panelPath.fill()

    let iconSize: CGFloat = W * 0.22   // ~145 px at 1x
    let iconY = (H - iconSize) / 2 + H * 0.04

    // ── Talkback app icon (left) ──────────────────────────────────────────────
    if let appImg = NSImage(contentsOfFile: appIconPath) {
        let iconX = W * 0.20 - iconSize / 2
        appImg.draw(in: NSRect(x: iconX, y: iconY, width: iconSize, height: iconSize),
                    from: .zero, operation: .sourceOver, fraction: 1.0)

        // "Talkback" label under icon
        let label = NSAttributedString(string: "Talkback", attributes: [
            .font: NSFont.systemFont(ofSize: W * 0.026, weight: .medium),
            .foregroundColor: NSColor(white: 0.25, alpha: 1.0),
        ])
        let ls = label.size()
        label.draw(at: NSPoint(x: W * 0.20 - ls.width / 2, y: iconY - ls.height - W * 0.012))
    }

    // ── arrow ────────────────────────────────────────────────────────────────
    let arrowAttrs: [NSAttributedString.Key: Any] = [
        .font: NSFont.systemFont(ofSize: W * 0.10, weight: .ultraLight),
        .foregroundColor: NSColor(white: 0.50, alpha: 0.85),
    ]
    let arrow = NSAttributedString(string: "→", attributes: arrowAttrs)
    let as_ = arrow.size()
    arrow.draw(at: NSPoint(x: (W - as_.width) / 2, y: (H - as_.height) / 2 + H * 0.04))

    // ── Applications folder icon (right) ─────────────────────────────────────
    let appsURL = URL(fileURLWithPath: "/Applications")
    let appsIcon = NSWorkspace.shared.icon(forFile: appsURL.path)
    let appsX = W * 0.80 - iconSize / 2
    appsIcon.draw(in: NSRect(x: appsX, y: iconY, width: iconSize, height: iconSize),
                  from: .zero, operation: .sourceOver, fraction: 1.0)

    // "Applications" label under folder icon
    let appsLabel = NSAttributedString(string: "Applications", attributes: [
        .font: NSFont.systemFont(ofSize: W * 0.026, weight: .medium),
        .foregroundColor: NSColor(white: 0.25, alpha: 1.0),
    ])
    let als = appsLabel.size()
    appsLabel.draw(at: NSPoint(x: W * 0.80 - als.width / 2, y: iconY - als.height - W * 0.012))

    // ── caption ──────────────────────────────────────────────────────────────
    let caption = NSAttributedString(string: "Drag Talkback to Applications to install", attributes: [
        .font: NSFont.systemFont(ofSize: W * 0.028, weight: .regular),
        .foregroundColor: NSColor(white: 0.38, alpha: 1.0),
    ])
    let cs = caption.size()
    caption.draw(at: NSPoint(x: (W - cs.width) / 2, y: H * 0.072))

    NSGraphicsContext.restoreGraphicsState()
    return rep
}

func save(rep: NSBitmapImageRep, to path: String) {
    guard let data = rep.representation(using: .png, properties: [:]) else {
        fputs("error: could not encode PNG for \(path)\n", stderr)
        exit(1)
    }
    do {
        try data.write(to: URL(fileURLWithPath: path))
    } catch {
        fputs("error: \(error)\n", stderr)
        exit(1)
    }
}

// ── main ─────────────────────────────────────────────────────────────────────

let args = CommandLine.arguments
guard args.count == 4 else {
    fputs("usage: swift make_dmg_bg.swift <app-icon.png> <out-1x.png> <out-2x.png>\n", stderr)
    exit(1)
}
let iconPath = args[1]
let out1x    = args[2]
let out2x    = args[3]

// 1x (660×400)
let rep1x = renderBackground(size: NSSize(width: 660, height: 400), appIconPath: iconPath)
save(rep: rep1x, to: out1x)

// 2x (1320×800) — Finder auto-selects this on Retina displays
let rep2x = renderBackground(size: NSSize(width: 1320, height: 800), appIconPath: iconPath)
save(rep: rep2x, to: out2x)

print("DMG background written: \(out1x), \(out2x)")
