package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"kaze/internal/config"
	"kaze/internal/jsonc"
	"kaze/internal/pathutil"
)

// WorkspaceInfo is what the frontend receives after opening or saving a
// workspace file. Folders are resolved absolute paths; Name is the file's base
// name without the extension, used as the workspace label.
type WorkspaceInfo struct {
	Path    string           `json:"path"`
	Name    string           `json:"name"`
	Folders []string         `json:"folders"`
	Session WorkspaceSession `json:"session"`
}

// workspaceFile is the on-disk format. Folders are relative to the workspace
// file's directory when possible, so the file can be committed with the
// project it describes.
type workspaceFile struct {
	Folders []string `json:"folders"`
}

// codeWorkspaceFile is the part of a VS Code .code-workspace file Kaze reads.
// Only folders are imported; settings, extensions, launch and tasks are
// ignored, and the file is never written back.
type codeWorkspaceFile struct {
	Folders []struct {
		Path string `json:"path"`
		URI  string `json:"uri"`
	} `json:"folders"`
}

const (
	workspaceExtension     = ".kaze-workspace"
	codeWorkspaceExtension = ".code-workspace"
	defaultWorkspace       = "workspace"
)

// OpenWorkspaceDialog asks for a workspace file and loads its folders. An empty
// WorkspaceInfo means the user cancelled, which is not an error.
func (a *App) OpenWorkspaceDialog() (WorkspaceInfo, error) {
	if a.wails == nil {
		return WorkspaceInfo{}, errors.New("workspace dialog is not available before startup")
	}
	path, err := a.wails.Dialog.OpenFile().
		SetTitle("Open Workspace").
		AddFilter("Workspace", "*"+workspaceExtension+";*"+codeWorkspaceExtension).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return WorkspaceInfo{}, err
	}
	return a.LoadWorkspace(path)
}

// LoadWorkspace reads a workspace file and attaches the tab set parked under
// it, so opening a workspace from the menu and restoring the last one at
// launch follow the same path. An empty session means this workspace has not
// been open since session.json was written.
func (a *App) LoadWorkspace(path string) (WorkspaceInfo, error) {
	info, err := loadWorkspaceFile(path)
	if err != nil {
		return WorkspaceInfo{}, err
	}
	info.Session = a.parkedWorkspace(info)
	return info, nil
}

// SaveWorkspaceAs asks where to write the current folder list and writes a
// workspace file there. The folders are stored relative to the file when they
// share a directory tree, so the result can live in the repository.
func (a *App) SaveWorkspaceAs(folders []string) (WorkspaceInfo, error) {
	if a.wails == nil {
		return WorkspaceInfo{}, errors.New("workspace dialog is not available before startup")
	}
	if len(config.NormalizeFolderList(folders)) == 0 {
		return WorkspaceInfo{}, errors.New("add a folder before saving a workspace")
	}

	dialog := a.wails.Dialog.SaveFile()
	dialog.SetOptions(&application.SaveFileDialogOptions{Title: "Save Workspace As"})
	dialog.AddFilter(appMeta.ProductName+" Workspace", "*"+workspaceExtension)
	dialog.SetFilename(workspaceSuggestedName(folders))
	path, err := dialog.PromptForSingleSelection()
	if err != nil || path == "" {
		return WorkspaceInfo{}, err
	}
	if !strings.HasSuffix(path, workspaceExtension) {
		path += workspaceExtension
	}
	return writeWorkspaceFile(path, folders)
}

// loadWorkspaceFile reads a workspace file and resolves its folders. Every
// folder must still exist; a missing one is reported instead of leaving a
// half-open workspace.
func loadWorkspaceFile(path string) (WorkspaceInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceInfo{}, fmt.Errorf("read workspace file: %w", err)
	}

	var entries []string
	if strings.EqualFold(filepath.Ext(path), codeWorkspaceExtension) {
		entries, err = parseCodeWorkspace(data)
	} else {
		var file workspaceFile
		err = json.Unmarshal(data, &file)
		entries = file.Folders
	}
	if err != nil {
		return WorkspaceInfo{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}

	dir := filepath.Dir(path)
	folders := make([]string, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		folder := filepath.FromSlash(entry)
		if !filepath.IsAbs(folder) {
			folder = filepath.Join(dir, folder)
		}
		folder = filepath.Clean(folder)
		info, err := os.Stat(folder)
		if err != nil {
			return WorkspaceInfo{}, fmt.Errorf("workspace folder %q: %w", entry, err)
		}
		if !info.IsDir() {
			return WorkspaceInfo{}, fmt.Errorf("workspace folder %q is not a folder", entry)
		}
		folders = append(folders, folder)
	}
	folders = pathutil.UniquePaths(folders)
	if len(folders) == 0 {
		return WorkspaceInfo{}, errors.New("workspace has no folders")
	}

	return WorkspaceInfo{
		Path:    path,
		Name:    workspaceName(path),
		Folders: folders,
	}, nil
}

// parseCodeWorkspace returns the folder entries of a VS Code workspace file.
// The file is JSONC, so comments and trailing commas are removed first. A
// file:// uri becomes its local path; any other scheme names a remote folder
// Kaze cannot open.
func parseCodeWorkspace(data []byte) ([]string, error) {
	var file codeWorkspaceFile
	if err := json.Unmarshal(jsonc.StripTrailingCommas(jsonc.StripComments(data)), &file); err != nil {
		return nil, err
	}
	entries := make([]string, 0, len(file.Folders))
	for _, folder := range file.Folders {
		if folder.Path != "" {
			entries = append(entries, folder.Path)
			continue
		}
		if folder.URI == "" {
			continue
		}
		u, err := url.Parse(folder.URI)
		if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") {
			return nil, fmt.Errorf("folder %q is not a local folder", folder.URI)
		}
		entries = append(entries, u.Path)
	}
	return entries, nil
}

// writeWorkspaceFile validates folders, encodes them relative to path's
// directory when they sit inside it, and replaces the file atomically.
func writeWorkspaceFile(path string, folders []string) (WorkspaceInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return WorkspaceInfo{}, errors.New("no workspace file")
	}

	absolute := make([]string, 0, len(folders))
	for _, folder := range config.NormalizeFolderList(folders) {
		folder, err := filepath.Abs(folder)
		if err != nil {
			return WorkspaceInfo{}, err
		}
		info, err := os.Stat(folder)
		if err != nil {
			return WorkspaceInfo{}, err
		}
		if !info.IsDir() {
			return WorkspaceInfo{}, fmt.Errorf("%s is not a folder", filepath.Base(folder))
		}
		absolute = append(absolute, folder)
	}
	absolute = pathutil.UniquePaths(absolute)
	if len(absolute) == 0 {
		return WorkspaceInfo{}, errors.New("add a folder before saving a workspace")
	}

	dir := filepath.Dir(path)
	entries := make([]string, 0, len(absolute))
	for _, folder := range absolute {
		relative, err := filepath.Rel(dir, folder)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			entries = append(entries, filepath.ToSlash(relative))
			continue
		}
		entries = append(entries, filepath.ToSlash(folder))
	}

	data, err := json.MarshalIndent(workspaceFile{Folders: entries}, "", "  ")
	if err != nil {
		return WorkspaceInfo{}, fmt.Errorf("encode workspace: %w", err)
	}
	data = append(data, '\n')
	if err := writeWorkspaceData(path, data); err != nil {
		return WorkspaceInfo{}, err
	}

	return WorkspaceInfo{
		Path:    path,
		Name:    workspaceName(path),
		Folders: absolute,
	}, nil
}

// writeWorkspaceData writes through a temporary file so a reader never sees a
// partially written workspace.
func writeWorkspaceData(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create workspace directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "workspace-*.json")
	if err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write workspace file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	return nil
}

// workspaceSuggestedName is the default file name in the Save As panel: the
// first folder's name, or "workspace" when there is none to borrow.
func workspaceSuggestedName(folders []string) string {
	for _, folder := range config.NormalizeFolderList(folders) {
		base := filepath.Base(folder)
		if base != "" && base != "." && base != string(filepath.Separator) {
			return base + workspaceExtension
		}
	}
	return defaultWorkspace + workspaceExtension
}

// workspaceName is the label shown for a workspace: the file name without its
// extension.
func workspaceName(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return defaultWorkspace
	}
	return name
}
