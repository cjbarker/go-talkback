#import <Cocoa/Cocoa.h>
#include "floatbutton.h"

// Go callbacks — implemented via //export in floatbutton.go.
extern void floatButtonPressed(void);
extern void floatButtonReleased(void);
extern void floatButtonMoved(double x, double y);

// ---------------------------------------------------------------------------
// TalkbackView — circular push-to-talk button drawn entirely in drawRect:.
// ---------------------------------------------------------------------------

@interface TalkbackView : NSView
@property (nonatomic, assign) BOOL isRecording;
@property (nonatomic, assign) BOOL isHovered;
@property (nonatomic, assign) BOOL isDragging;
@end

@implementation TalkbackView

- (instancetype)initWithFrame:(NSRect)frame {
    self = [super initWithFrame:frame];
    if (self) {
        // Track mouse hover so we can lighten the button on hover.
        NSTrackingArea *area = [[NSTrackingArea alloc]
            initWithRect:frame
                 options:NSTrackingMouseEnteredAndExited | NSTrackingActiveAlways
                   owner:self
                userInfo:nil];
        [self addTrackingArea:area];
    }
    return self;
}

- (void)drawRect:(NSRect)rect {
    NSRect bounds = self.bounds;
    // Inset so the shadow/clipping doesn't clip the circle edge.
    NSRect circle = NSInsetRect(bounds, 4, 4);

    // Draw drop shadow.
    [NSGraphicsContext saveGraphicsState];
    NSShadow *shadow = [[NSShadow alloc] init];
    shadow.shadowColor     = [NSColor colorWithWhite:0.0 alpha:0.45];
    shadow.shadowOffset    = NSMakeSize(0, -2);
    shadow.shadowBlurRadius = 8;
    [shadow set];

    // Fill background circle.
    NSColor *bg;
    if (self.isRecording) {
        bg = [NSColor colorWithRed:0.85 green:0.12 blue:0.12 alpha:1.0];
    } else if (self.isHovered) {
        bg = [NSColor colorWithWhite:0.38 alpha:0.95];
    } else {
        bg = [NSColor colorWithWhite:0.22 alpha:0.92];
    }
    [bg setFill];
    [[NSBezierPath bezierPathWithOvalInRect:circle] fill];
    [NSGraphicsContext restoreGraphicsState];

    // Draw a thin border ring when recording.
    if (self.isRecording) {
        [[NSColor colorWithWhite:1.0 alpha:0.35] setStroke];
        NSBezierPath *ring = [NSBezierPath bezierPathWithOvalInRect:NSInsetRect(circle, 2, 2)];
        ring.lineWidth = 1.5;
        [ring stroke];
    }

    // Microphone emoji label.
    NSString *label    = @"🎤";
    NSFont   *font     = [NSFont systemFontOfSize:26];
    NSDictionary *attrs = @{NSFontAttributeName: font};
    NSSize   textSize  = [label sizeWithAttributes:attrs];
    NSPoint  textPt    = NSMakePoint(
        (bounds.size.width  - textSize.width)  / 2,
        (bounds.size.height - textSize.height) / 2
    );
    [label drawAtPoint:textPt withAttributes:attrs];
}

// Accept the first mouse-down without requiring the panel to become active first.
- (BOOL)acceptsFirstMouse:(NSEvent *)event { return YES; }

- (void)mouseDown:(NSEvent *)event {
    self.isDragging = NO;
    floatButtonPressed();
}

- (void)mouseUp:(NSEvent *)event {
    if (self.isDragging) {
        // Report the final position so Go can persist it.
        NSPoint origin = self.window.frame.origin;
        floatButtonMoved(origin.x, origin.y);
        self.isDragging = NO;
    } else {
        floatButtonReleased();
    }
}

// Drag the window by moving it while the button is held.
- (void)mouseDragged:(NSEvent *)event {
    self.isDragging = YES;
    NSWindow *win = self.window;
    NSPoint origin = win.frame.origin;
    origin.x += event.deltaX;
    origin.y -= event.deltaY;
    [win setFrameOrigin:origin];
}

- (void)mouseEntered:(NSEvent *)event {
    self.isHovered = YES;
    [self setNeedsDisplay:YES];
}

- (void)mouseExited:(NSEvent *)event {
    self.isHovered = NO;
    [self setNeedsDisplay:YES];
}

@end

// ---------------------------------------------------------------------------
// Panel management
// ---------------------------------------------------------------------------

static NSPanel    *gPanel = nil;
static TalkbackView *gView  = nil;

// clampToScreen: returns a frame origin adjusted so that the w×h panel is
// fully inside some screen's visible area. Falls back to centering on the
// main screen when the point is off all screens.
static NSPoint clampToScreen(CGFloat x, CGFloat y, CGFloat w, CGFloat h) {
    NSRect proposed = NSMakeRect(x, y, w, h);
    // Check every screen; use the one whose visible frame intersects most.
    for (NSScreen *screen in [NSScreen screens]) {
        NSRect vis = screen.visibleFrame;
        if (NSIntersectsRect(proposed, vis)) {
            // Clamp so the panel stays fully inside this screen.
            CGFloat cx = MAX(vis.origin.x, MIN(x, NSMaxX(vis) - w));
            CGFloat cy = MAX(vis.origin.y, MIN(y, NSMaxY(vis) - h));
            return NSMakePoint(cx, cy);
        }
    }
    // Off all screens: center on the main screen.
    NSRect vis = [NSScreen mainScreen].visibleFrame;
    return NSMakePoint(NSMidX(vis) - w / 2, NSMidY(vis) - h / 2);
}

// createPanel: builds the NSPanel at the given frame origin. Must be called on
// the main thread.
static void createPanel(CGFloat x, CGFloat y) {
    CGFloat w = 80, h = 80;
    NSRect frame = NSMakeRect(x, y, w, h);

    // NSWindowStyleMaskNonactivatingPanel prevents the panel from
    // stealing keyboard focus from the app the user is typing in.
    NSWindowStyleMask mask =
        NSWindowStyleMaskBorderless |
        NSWindowStyleMaskNonactivatingPanel;

    gPanel = [[NSPanel alloc]
        initWithContentRect:frame
                  styleMask:mask
                    backing:NSBackingStoreBuffered
                      defer:NO];

    gPanel.level           = NSFloatingWindowLevel;
    gPanel.opaque          = NO;
    gPanel.backgroundColor = [NSColor clearColor];
    gPanel.hasShadow       = NO; // drawn manually in TalkbackView

    // Don't show in Mission Control or Cmd+Tab.
    [gPanel setCollectionBehavior:
        NSWindowCollectionBehaviorCanJoinAllSpaces  |
        NSWindowCollectionBehaviorStationary        |
        NSWindowCollectionBehaviorIgnoresCycle];

    // Don't become key window on click.
    [gPanel setBecomesKeyOnlyIfNeeded:YES];

    gView = [[TalkbackView alloc] initWithFrame:NSMakeRect(0, 0, w, h)];
    [gPanel setContentView:gView];
}

void showFloatButton(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        CGFloat w = 80, h = 80;
        if (gPanel != nil) {
            // Re-center if the panel has drifted off all screens (e.g. after
            // a display configuration change removed the monitor it was on).
            NSPoint origin = gPanel.frame.origin;
            NSPoint safe   = clampToScreen(origin.x, origin.y, w, h);
            if (!NSEqualPoints(origin, safe)) {
                [gPanel setFrameOrigin:safe];
            }
            [gPanel orderFront:nil];
            return;
        }
        // First show with no saved position: center on the main screen.
        NSScreen *screen = [NSScreen mainScreen];
        NSRect    vis    = screen.visibleFrame;
        createPanel(NSMidX(vis) - w / 2, NSMidY(vis) - h / 2);
        [gPanel orderFront:nil];
    });
}

void showFloatButtonAt(double x, double y) {
    dispatch_async(dispatch_get_main_queue(), ^{
        CGFloat w = 80, h = 80;
        NSPoint safe = clampToScreen((CGFloat)x, (CGFloat)y, w, h);
        if (gPanel != nil) {
            [gPanel setFrameOrigin:safe];
            [gPanel orderFront:nil];
            return;
        }
        createPanel(safe.x, safe.y);
        [gPanel orderFront:nil];
    });
}

void hideFloatButton(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gPanel orderOut:nil];
    });
}

void setFloatButtonRecording(int active) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gPanel == nil) return;
        gView.isRecording = (active != 0);
        [gView setNeedsDisplay:YES];
    });
}
