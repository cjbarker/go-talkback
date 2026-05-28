#ifndef DIALOG_H
#define DIALOG_H

// showDownloadConfirmDialog displays a native NSAlert asking the user whether
// to download the base.en model. Returns 1 if the user clicks "Download", 0
// if they click "Quit". Must be called from the main thread or will
// dispatch_sync there automatically.
extern int showDownloadConfirmDialog(void);

// showProgressWindow opens a floating progress panel for the model download.
extern void showProgressWindow(void);

// updateProgressBar updates the progress indicator (0–100) and status label.
// Safe to call from any thread.
extern void updateProgressBar(int pct);

// closeProgressWithSuccess closes the progress panel and shows a success alert
// telling the user the app is ready. Blocks until the user clicks OK.
extern void closeProgressWithSuccess(void);

// closeProgressWindow closes the progress panel without a success message
// (used on download failure).
extern void closeProgressWindow(void);

// showLaunchAtLoginDialog asks the user whether to enable launch at login.
// Returns 1 if the user clicks "Enable", 0 if they click "Not Now".
extern int showLaunchAtLoginDialog(void);

#endif
