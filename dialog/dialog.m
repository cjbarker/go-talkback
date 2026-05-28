#import <AppKit/AppKit.h>
#include "dialog.h"

// ---------------------------------------------------------------------------
// Confirm dialog
// ---------------------------------------------------------------------------

int showDownloadConfirmDialog(void) {
    // NSAlert must run on the main thread. onReady() is called from a
    // goroutine by systray, so we dispatch_sync onto the main queue.
    __block NSModalResponse response = NSAlertSecondButtonReturn; // default = Quit

    dispatch_sync(dispatch_get_main_queue(), ^{
        NSAlert *alert = [[NSAlert alloc] init];
        [alert setMessageText:@"No Speech Model Found"];
        [alert setInformativeText:
            @"go-talkback needs a speech recognition model to function.\n\n"
            @"Would you like to download the base.en model (~142 MB) now? "
            @"This is the fastest model and works well for clear speech.\n\n"
            @"Download progress will be shown in a window and in the menu bar."];
        [alert addButtonWithTitle:@"Download"];
        [alert addButtonWithTitle:@"Quit"];
        [alert setAlertStyle:NSAlertStyleInformational];

        NSImage *icon = [NSImage imageNamed:@"NSApplicationIcon"];
        if (icon) {
            [alert setIcon:icon];
        }

        response = [alert runModal];
    });

    return (response == NSAlertFirstButtonReturn) ? 1 : 0;
}

// ---------------------------------------------------------------------------
// Progress window
// ---------------------------------------------------------------------------

static NSPanel            *progressPanel = nil;
static NSProgressIndicator *progressBar  = nil;
static NSTextField         *progressPct  = nil;

void showProgressWindow(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        // Panel frame (content area, excluding title bar).
        NSRect frame = NSMakeRect(0, 0, 420, 100);
        progressPanel = [[NSPanel alloc]
            initWithContentRect:frame
                      styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskNonactivatingPanel
                        backing:NSBackingStoreBuffered
                          defer:NO];
        [progressPanel setTitle:@"Downloading Model"];
        [progressPanel setFloatingPanel:YES];
        [progressPanel setReleasedWhenClosed:NO];
        [progressPanel center];

        NSView *cv = [progressPanel contentView];

        // Status label — top of panel.
        NSTextField *statusLabel = [[NSTextField alloc]
            initWithFrame:NSMakeRect(20, 68, 380, 18)];
        [statusLabel setStringValue:@"Downloading base.en (~142 MB)\u2026"];
        [statusLabel setBezeled:NO];
        [statusLabel setDrawsBackground:NO];
        [statusLabel setEditable:NO];
        [statusLabel setSelectable:NO];
        [cv addSubview:statusLabel];

        // Progress bar — middle.
        progressBar = [[NSProgressIndicator alloc]
            initWithFrame:NSMakeRect(20, 44, 380, 16)];
        [progressBar setStyle:NSProgressIndicatorStyleBar];
        [progressBar setIndeterminate:NO];
        [progressBar setMinValue:0.0];
        [progressBar setMaxValue:100.0];
        [progressBar setDoubleValue:0.0];
        [cv addSubview:progressBar];

        // Percentage label — below bar.
        progressPct = [[NSTextField alloc]
            initWithFrame:NSMakeRect(20, 14, 380, 18)];
        [progressPct setStringValue:@"Starting\u2026"];
        [progressPct setBezeled:NO];
        [progressPct setDrawsBackground:NO];
        [progressPct setEditable:NO];
        [progressPct setSelectable:NO];
        [progressPct setTextColor:[NSColor secondaryLabelColor]];
        [progressPct setFont:[NSFont systemFontOfSize:11.0]];
        [cv addSubview:progressPct];

        [progressPanel orderFront:nil];
    });
}

void updateProgressBar(int pct) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (!progressPanel) return;
        [progressBar setDoubleValue:(double)pct];
        [progressPct setStringValue:
            [NSString stringWithFormat:@"%d%% complete", pct]];
    });
}

void closeProgressWithSuccess(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        if (progressPanel) {
            [progressPanel close];
            progressPanel = nil;
            progressBar   = nil;
            progressPct   = nil;
        }

        // Bring the app forward so the alert is visible.
        [NSApp activateIgnoringOtherApps:YES];

        NSAlert *alert = [[NSAlert alloc] init];
        [alert setMessageText:@"Model Downloaded Successfully"];
        [alert setInformativeText:
            @"The base.en model is ready.\n\n"
            @"go-talkback is now ready to use — hold Option\u2060+\u2060Space "
            @"to start dictating."];
        [alert addButtonWithTitle:@"Get Started"];
        [alert setAlertStyle:NSAlertStyleInformational];

        NSImage *icon = [NSImage imageNamed:@"NSApplicationIcon"];
        if (icon) {
            [alert setIcon:icon];
        }

        [alert runModal];
    });
}

void closeProgressWindow(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (progressPanel) {
            [progressPanel close];
            progressPanel = nil;
            progressBar   = nil;
            progressPct   = nil;
        }
    });
}

// ---------------------------------------------------------------------------
// Launch-at-login dialog
// ---------------------------------------------------------------------------

int showLaunchAtLoginDialog(void) {
    __block NSModalResponse response = NSAlertSecondButtonReturn; // default = Not Now

    dispatch_sync(dispatch_get_main_queue(), ^{
        NSAlert *alert = [[NSAlert alloc] init];
        [alert setMessageText:@"Launch Talkback at Login?"];
        [alert setInformativeText:
            @"Talkback can start automatically each time you log in, "
            @"so it's always ready when you need it.\n\n"
            @"You can change this later from the menu bar."];
        [alert addButtonWithTitle:@"Enable"];
        [alert addButtonWithTitle:@"Not Now"];
        [alert setAlertStyle:NSAlertStyleInformational];

        NSImage *icon = [NSImage imageNamed:@"NSApplicationIcon"];
        if (icon) {
            [alert setIcon:icon];
        }

        response = [alert runModal];
    });

    return (response == NSAlertFirstButtonReturn) ? 1 : 0;
}
