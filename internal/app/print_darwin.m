#import <Cocoa/Cocoa.h>

// The print panel is AppKit's; these functions only build the view it prints.
// They are called on the main thread from print_darwin.go.

// pageRect is the area one page can hold: the chosen paper minus the
// imageable margins, so text never runs into the unprintable edge.
static NSRect pageRect(NSPrintInfo *info) {
    NSRect area = [info imageablePageBounds];
    if (area.size.width < 1 || area.size.height < 1) {
        // No usable print info yet: fall back to US Letter inside one-inch
        // margins, which is what AppKit defaults to.
        area = NSMakeRect(0, 0, 468, 648);
    }
    return area;
}

// runPrint shows the panel and starts printing. Running as a sheet keeps
// the webview responsive while the user picks a printer.
static void runPrint(NSView *view, NSPrintInfo *info) {
    NSPrintOperation *operation = [NSPrintOperation printOperationWithView:view printInfo:info];
    [operation setShowsPrintPanel:YES];
    [operation setShowsProgressPanel:YES];

    NSWindow *window = [NSApp keyWindow] ?: [NSApp mainWindow];
    if (window != nil) {
        [operation runOperationModalForWindow:window delegate:nil didRunSelector:NULL contextInfo:NULL];
    } else {
        [operation runOperation];
    }
}

// nativePrintText prints plain text. NSTextView paginates the text itself: a long
// document becomes several pages, a short one stays on a single sheet.
void nativePrintText(const char *text, const char *fontName, double fontSize) {
    @autoreleasepool {
        NSPrintInfo *info = [[NSPrintInfo sharedPrintInfo] copy];
        [info setHorizontalPagination:NSPrintingPaginationModeFit];
        [info setVerticalPagination:NSPrintingPaginationModeAutomatic];

        NSRect area = pageRect(info);
        NSTextView *view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, area.size.width, area.size.height)];
        [view setEditable:NO];
        [view setRichText:NO];
        [view setHorizontallyResizable:NO];
        [view setVerticallyResizable:YES];
        [[view textContainer] setWidthTracksTextView:YES];
        [[view textContainer] setContainerSize:NSMakeSize(area.size.width, CGFLOAT_MAX)];

        NSFont *font = nil;
        if (fontName != NULL && fontName[0] != '\0') {
            font = [NSFont fontWithName:[NSString stringWithUTF8String:fontName] size:fontSize];
        }
        if (font == nil) {
            font = [NSFont userFixedPitchFontOfSize:fontSize];
        }
        [view setFont:font];
        [view setString:(text != NULL ? [NSString stringWithUTF8String:text] : @"")];

        runPrint(view, info);
    }
}

// nativePrintImage prints a picture on one page, scaled to fit and centered.
// Returns 0 when the file cannot be decoded, so Go can report why.
int nativePrintImage(const char *path) {
    @autoreleasepool {
        NSImage *image = path != NULL ? [[NSImage alloc] initWithContentsOfFile:[NSString stringWithUTF8String:path]] : nil;
        if (image == nil) {
            return 0;
        }

        NSPrintInfo *info = [[NSPrintInfo sharedPrintInfo] copy];
        [info setHorizontalPagination:NSPrintingPaginationModeFit];
        [info setVerticalPagination:NSPrintingPaginationModeFit];

        NSImageView *view = [[NSImageView alloc] initWithFrame:pageRect(info)];
        [view setImage:image];
        [view setImageScaling:NSImageScaleProportionallyUpOrDown];
        [view setImageAlignment:NSImageAlignCenter];

        runPrint(view, info);
        return 1;
    }
}
