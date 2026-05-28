-- dmg_layout.applescript
-- Configures the Finder window appearance for the Talkback DMG installer.
-- Run after mounting the DMG and copying the .background/ folder.
-- The .DS_Store created here is preserved in the final read-only DMG.

tell application "Finder"
    tell disk "Talkback"
        open

        -- Use icon view with no toolbar or status bar.
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false

        -- Window dimensions match the background image (660×400).
        set the bounds of container window to {300, 100, 960, 500}

        -- Icon view options.
        set viewOptions to the icon view options of container window
        set arrangement of viewOptions to not arranged
        set icon size of viewOptions to 128
        set text size of viewOptions to 13
        set background picture of viewOptions to ¬
            POSIX file "/Volumes/Talkback/.background/background.png"

        -- Position icons: Talkback.app left, Applications alias right.
        -- Coordinates are (x, y) from top-left of the Finder window content area.
        set position of item "Talkback.app" of container window to {155, 195}
        set position of item "Applications" of container window to {505, 195}

        -- Flush Finder state so .DS_Store is written.
        close
        open
        update without registering applications
        delay 3
        close
    end tell
end tell
