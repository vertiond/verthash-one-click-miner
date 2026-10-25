package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevealTargetExistingFileIsSelected(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "vertcoin.conf")
	if err := os.WriteFile(file, []byte("server=1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	target, selectIt := revealTarget(file)
	if target != file || !selectIt {
		t.Errorf("revealTarget = %q, %v; want the file, selected", target, selectIt)
	}
}

func TestRevealTargetMissingFileOpensItsFolder(t *testing.T) {
	dir := t.TempDir()
	target, selectIt := revealTarget(filepath.Join(dir, "vertcoin.conf"))
	if target != dir || selectIt {
		t.Errorf("revealTarget = %q, %v; want the folder, not selected", target, selectIt)
	}
}

func TestRevealTargetMissingFolderOpensNearestExisting(t *testing.T) {
	dir := t.TempDir()
	target, selectIt := revealTarget(filepath.Join(dir, "Vertcoin", "vertcoin.conf"))
	if target != dir || selectIt {
		t.Errorf("revealTarget = %q, %v; want %q, not selected", target, selectIt, dir)
	}
}

func TestExplorerCmdLine(t *testing.T) {
	got := explorerCmdLine(`C:\Users\Jane Doe\AppData\Roaming\Vertcoin\vertcoin.conf`, true)
	want := `explorer.exe /select,"C:\Users\Jane Doe\AppData\Roaming\Vertcoin\vertcoin.conf"`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	got = explorerCmdLine(`C:\Users\Jane Doe\AppData\Roaming\Vertcoin`, false)
	want = `explorer.exe "C:\Users\Jane Doe\AppData\Roaming\Vertcoin"`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
