//go:build darwin && cgo

#import <Cocoa/Cocoa.h>

// nativeTrashPath moves one file or folder to the user's Trash. It returns 1 on
// success; otherwise *failure is a malloc'd message the Go side frees.

int nativeTrashPath(const char *path, char **failure) {
    @autoreleasepool {
        if (path == NULL) {
            if (failure != NULL) {
                *failure = strdup("no path to move to the Trash");
            }
            return 0;
        }
        NSFileManager *manager = [NSFileManager defaultManager];
        NSString *file = [NSString stringWithUTF8String:path];
        if (file == nil) {
            if (failure != NULL) {
                *failure = strdup("the path is not valid UTF-8");
            }
            return 0;
        }

        NSURL *url = nil;
        NSError *error = nil;
        if (![manager trashItemAtURL:[NSURL fileURLWithPath:file] resultingItemURL:&url error:&error]) {
            if (failure != NULL) {
                NSString *message = error != nil ? [error localizedDescription]
                                                 : @"the file manager refused the request";
                *failure = strdup([message UTF8String]);
            }
            return 0;
        }
        return 1;
    }
}
