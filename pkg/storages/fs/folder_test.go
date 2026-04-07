package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g/pkg/storages/storage"
)

func TestFSFolder(t *testing.T) {
	tmpDir := setupTmpDir(t)

	defer os.RemoveAll(tmpDir)

	st, err := ConfigureStorage(tmpDir, nil)
	assert.NoError(t, err)

	storage.RunFolderTest(st.RootFolder(), t)
}

func TestDeleteObjectsRemovesEmptyFolders(t *testing.T) {
	tmpDir := setupTmpDir(t)
	defer os.RemoveAll(tmpDir)

	st, err := ConfigureStorage(tmpDir, nil)
	assert.NoError(t, err)

	root := st.RootFolder()

	// Create a nested structure: root/a/b/file
	sub := root.GetSubFolder("a/b")
	err = sub.PutObject("file", strings.NewReader("data"))
	assert.NoError(t, err)

	// Verify the directory exists
	_, err = os.Stat(filepath.Join(tmpDir, "a", "b"))
	assert.NoError(t, err)

	// Delete the file
	err = sub.DeleteObjects([]storage.Object{storage.NewLocalObject("file", time.Time{}, 0)})
	assert.NoError(t, err)

	// Both empty subdirectories should have been cleaned up
	_, err = os.Stat(filepath.Join(tmpDir, "a", "b"))
	assert.True(t, os.IsNotExist(err), "empty subdirectory 'b' should have been removed")

	_, err = os.Stat(filepath.Join(tmpDir, "a"))
	assert.True(t, os.IsNotExist(err), "empty subdirectory 'a' should have been removed")

	// Root directory must still exist
	_, err = os.Stat(tmpDir)
	assert.NoError(t, err, "root directory must not be removed")
}

func TestDeleteObjectsKeepsNonEmptyFolders(t *testing.T) {
	tmpDir := setupTmpDir(t)
	defer os.RemoveAll(tmpDir)

	st, err := ConfigureStorage(tmpDir, nil)
	assert.NoError(t, err)

	root := st.RootFolder()

	// Create two files in the same subdirectory
	sub := root.GetSubFolder("a/b")
	err = sub.PutObject("file1", strings.NewReader("data1"))
	assert.NoError(t, err)
	err = sub.PutObject("file2", strings.NewReader("data2"))
	assert.NoError(t, err)

	// Delete only one file
	err = sub.DeleteObjects([]storage.Object{storage.NewLocalObject("file1", time.Time{}, 0)})
	assert.NoError(t, err)

	// The subdirectory should still exist because file2 is still there
	_, err = os.Stat(filepath.Join(tmpDir, "a", "b"))
	assert.NoError(t, err, "non-empty subdirectory 'b' should not be removed")
}

func setupTmpDir(t *testing.T) string {
	cwd, err := filepath.Abs("./")
	if err != nil {
		t.Log(err)
	}
	// Create temp directory.
	tmpDir, err := os.MkdirTemp(cwd, "data")
	if err != nil {
		t.Log(err)
	}
	err = os.Chmod(tmpDir, 0755)
	if err != nil {
		t.Log(err)
	}
	return tmpDir
}
