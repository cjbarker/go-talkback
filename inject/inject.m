#import <CoreGraphics/CoreGraphics.h>
#import <Foundation/Foundation.h>
#include "inject.h"

// injectText posts Unicode characters to the HID event tap, which types them
// into whatever application currently has keyboard focus. This bypasses keyboard
// layout mapping so non-ASCII characters work correctly in all apps.
void injectText(const char *utf8) {
    if (!utf8 || utf8[0] == '\0') return;

    NSString *str = [NSString stringWithUTF8String:utf8];
    if (!str) return;

    NSUInteger len = [str length];
    if (len == 0) return;

    // Process in chunks to avoid stack overflow on very long strings.
    const NSUInteger chunkSize = 20;
    for (NSUInteger i = 0; i < len; i += chunkSize) {
        NSUInteger remaining = len - i;
        NSUInteger count = remaining < chunkSize ? remaining : chunkSize;

        UniChar buf[20]; // fixed size == chunkSize
        [str getCharacters:buf range:NSMakeRange(i, count)];

        // keyDown + keyUp pair; keyCode 0 is unused since we set the unicode string directly.
        CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, 0, true);
        CGEventRef keyUp   = CGEventCreateKeyboardEvent(NULL, 0, false);
        CGEventKeyboardSetUnicodeString(keyDown, count, buf);
        CGEventKeyboardSetUnicodeString(keyUp,   count, buf);
        CGEventPost(kCGHIDEventTap, keyDown);
        CGEventPost(kCGHIDEventTap, keyUp);
        CFRelease(keyDown);
        CFRelease(keyUp);
    }
}
