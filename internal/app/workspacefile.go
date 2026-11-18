package app

import (
	"errors"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"kaze/internal/config"
	"kaze/internal/workspacefile"
)

// OpenWorkspaceDialog asks for a workspace file and loads its folders. An empty
// WorkspaceInfo means the user cancelled, which is not an error.
func (a *App) OpenWorkspaceDialog() (workspacefile.WorkspaceInfo, error) {
	if a.wails == nil {
		return workspacefile.WorkspaceInfo{}, errors.New("workspace dialog is not available before startup")
	}
	path, err := a.wails.Dialog.OpenFile().
		SetTitle("Open Workspace").
		AddFilter("Workspace", "*"+workspacefile.Extension+";*"+workspacefile.CodeExtension).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return workspacefile.WorkspaceInfo{}, err
	}
	return a.LoadWorkspace(path)
}

// LoadWorkspace reads a workspace file and attaches the tab set parked under
// it, so opening a workspace from the menu and restoring the last one at
// launch follow the same path. An empty session means this workspace has not
// been open since session.json was written.
func (a *App) LoadWorkspace(path string) (workspacefile.WorkspaceInfo, error) {
	info, err := workspacefile.Load(path)
	if err != nil {
		return workspacefile.WorkspaceInfo{}, err
	}
	info.Session = a.parkedWorkspace(info)
	return info, nil
}

// SaveWorkspaceAs asks where to write the current folder list and writes a
// workspace file there. The folders are stored relative to the file when they
// share a directory tree, so the result can live in the repository.
func (a *App) SaveWorkspaceAs(folders []string) (workspacefile.WorkspaceInfo, error) {
	if a.wails == nil {
		return workspacefile.WorkspaceInfo{}, errors.New("workspace dialog is not available before startup")
	}
	if len(config.NormalizeFolderList(folders)) == 0 {
		return workspacefile.WorkspaceInfo{}, errors.New("add a folder before saving a workspace")
	}

	dialog := a.wails.Dialog.SaveFile()
	dialog.SetOptions(&application.SaveFileDialogOptions{Title: "Save Workspace As"})
	dialog.AddFilter(appMeta.ProductName+" Workspace", "*"+workspacefile.Extension)
	dialog.SetFilename(workspacefile.SuggestedName(folders))
	path, err := dialog.PromptForSingleSelection()
	if err != nil || path == "" {
		return workspacefile.WorkspaceInfo{}, err
	}
	if !strings.HasSuffix(path, workspacefile.Extension) {
		path += workspacefile.Extension
	}
	return workspacefile.Write(path, folders)
}
