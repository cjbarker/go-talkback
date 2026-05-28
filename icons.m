#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// renderBitmapRep creates a transparent RGBA bitmap rep of the given size.
static NSBitmapImageRep *makeBitmapRep(int size) {
    return [[NSBitmapImageRep alloc]
        initWithBitmapDataPlanes:NULL
                      pixelsWide:size
                      pixelsHigh:size
                   bitsPerSample:8
                 samplesPerPixel:4
                        hasAlpha:YES
                        isPlanar:NO
                  colorSpaceName:NSCalibratedRGBColorSpace
                     bytesPerRow:0
                    bitsPerPixel:0];
}

// drawEmojiInContext draws the emoji string centred in a size×size context.
static void drawEmoji(NSString *str, int size) {
    NSFont       *font = [NSFont systemFontOfSize:size * 0.78];
    NSDictionary *attrs = @{NSFontAttributeName: font};
    NSSize        ts   = [str sizeWithAttributes:attrs];
    NSPoint       pt   = NSMakePoint((size - ts.width)  / 2.0,
                                     (size - ts.height) / 2.0);
    [str drawAtPoint:pt withAttributes:attrs];
}

// repToPNG encodes a bitmap rep as PNG and returns a malloc'd buffer.
static unsigned char *repToPNG(NSBitmapImageRep *rep, int *outLen) {
    NSData        *png = [rep representationUsingType:NSBitmapImageFileTypePNG
                                           properties:@{}];
    unsigned char *buf = malloc([png length]);
    memcpy(buf, [png bytes], [png length]);
    *outLen = (int)[png length];
    return buf;
}

// Renders an emoji string into a square PNG of the given pixel size.
// Returns a malloc'd buffer; *outLen is set to its byte length.
// The caller must free() the returned pointer.
unsigned char *renderEmojiPNG(const char *emoji, int size, int *outLen) {
    NSBitmapImageRep *rep = makeBitmapRep(size);

    [NSGraphicsContext saveGraphicsState];
    [NSGraphicsContext setCurrentContext:
        [NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];

    [[NSColor clearColor] set];
    NSRectFill(NSMakeRect(0, 0, size, size));
    drawEmoji([NSString stringWithUTF8String:emoji], size);

    [NSGraphicsContext restoreGraphicsState];
    return repToPNG(rep, outLen);
}

// Renders the emoji tinted solid red (opacity 0.85) by compositing a red
// rectangle over the emoji pixels using NSCompositeSourceAtop — every opaque
// pixel keeps its alpha but adopts the red colour.
unsigned char *renderEmojiRedPNG(const char *emoji, int size, int *outLen) {
    NSBitmapImageRep *rep = makeBitmapRep(size);

    [NSGraphicsContext saveGraphicsState];
    [NSGraphicsContext setCurrentContext:
        [NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];

    [[NSColor clearColor] set];
    NSRectFill(NSMakeRect(0, 0, size, size));
    drawEmoji([NSString stringWithUTF8String:emoji], size);

    // Flood the non-transparent pixels with red, preserving per-pixel alpha.
    [[NSColor colorWithRed:0.85 green:0.0 blue:0.0 alpha:0.90] set];
    NSRectFillUsingOperation(NSMakeRect(0, 0, size, size),
                             NSCompositingOperationSourceAtop);

    [NSGraphicsContext restoreGraphicsState];
    return repToPNG(rep, outLen);
}
