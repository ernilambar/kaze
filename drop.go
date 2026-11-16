package main

// dropLaunchRequest turns the paths a native file drop reported into the same
// request a command line launch uses. Files open as tabs and folders become
// sidebar roots, because the frontend handles both through applyLaunch. With
// newWindow the drop goes to a window of its own, and a dropped folder becomes
// that window's only root, matching File → Open Folder in New Window. A drop
// that names no file or folder has nothing to open, so holding Option over it
// does not create an empty window.
func dropLaunchRequest(paths []string, newWindow bool) LaunchRequest {
	request := parseLaunchRequest(paths)
	if !newWindow || (len(request.Files) == 0 && len(request.Folders) == 0) {
		return request
	}
	request.NewWindow = true
	if len(request.Folders) > 0 {
		request.ReplaceFolders = true
	}
	return request
}

// handleFileDrop routes one window's dropped paths through the launch path, so
// its frontend opens files as tabs and adds folders as roots. The window is the
// one the drop landed in, not the focused one, because a drag can target a
// window behind the frontmost one.
func (a *App) handleFileDrop(session *windowSession, paths []string, newWindow bool) {
	request := dropLaunchRequest(paths, newWindow)
	if request.Empty() {
		return
	}
	a.queueLaunchTo(session, request)
}
