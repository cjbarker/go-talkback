#pragma once

// Show the floating push-to-talk button at screen coords (x, y), creating it
// if needed. Pass x=0, y=0 to use the default centered position.
void showFloatButton(void);
void showFloatButtonAt(double x, double y);

// Hide the floating button (does not destroy it).
void hideFloatButton(void);

// Update the button's visual recording state (1 = recording, 0 = idle).
void setFloatButtonRecording(int active);
