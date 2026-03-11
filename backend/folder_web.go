//go:build !wails

package backend

import "context"

func OpenFolderInExplorer(_ string) error                              { return nil }
func SelectFolderDialog(_ context.Context, _ string) (string, error)  { return "", nil }
func SelectFileDialog(_ context.Context) (string, error)               { return "", nil }
func SelectImageVideoDialog(_ context.Context) ([]string, error)       { return nil, nil }
