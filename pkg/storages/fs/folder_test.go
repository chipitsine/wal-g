package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestEmptyFolderCleanup(t *testing.T) {
	tmpDir := setupTmpDir(t)
	defer os.RemoveAll(tmpDir)

	// Create a folder with nested structure
	folder := NewFolder(tmpDir, "")
	
	// Create nested directories with files
	// Structure: tmpDir/base_backups_005/base_0001/tar_partitions/file.txt
	nestedPath := "base_backups_005/base_0001/tar_partitions/file.txt"
	err := folder.PutObject(nestedPath, strings.NewReader("test content"))
	assert.NoError(t, err)
	
	// Verify the directory structure exists
	dirPath := filepath.Join(tmpDir, "base_backups_005", "base_0001", "tar_partitions")
	_, err = os.Stat(dirPath)
	assert.NoError(t, err, "Directory should exist before deletion")
	
	// Delete the file
	err = folder.DeleteObjects([]string{nestedPath})
	assert.NoError(t, err)
	
	// Verify that empty parent directories are removed
	// tar_partitions should be removed
	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err), "tar_partitions directory should be removed")
	
	// base_0001 should be removed
	dirPath = filepath.Join(tmpDir, "base_backups_005", "base_0001")
	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err), "base_0001 directory should be removed")
	
	// base_backups_005 should be removed
	dirPath = filepath.Join(tmpDir, "base_backups_005")
	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err), "base_backups_005 directory should be removed")
	
	// Root tmpDir should still exist
	_, err = os.Stat(tmpDir)
	assert.NoError(t, err, "Root directory should still exist")
}

func TestEmptyFolderCleanupPartial(t *testing.T) {
	tmpDir := setupTmpDir(t)
	defer os.RemoveAll(tmpDir)

	folder := NewFolder(tmpDir, "")
	
	// Create two files in the same parent directory
	// Structure: 
	//   tmpDir/base_backups_005/base_0001/file1.txt
	//   tmpDir/base_backups_005/base_0001/file2.txt
	file1 := "base_backups_005/base_0001/file1.txt"
	file2 := "base_backups_005/base_0001/file2.txt"
	
	err := folder.PutObject(file1, strings.NewReader("content1"))
	assert.NoError(t, err)
	err = folder.PutObject(file2, strings.NewReader("content2"))
	assert.NoError(t, err)
	
	// Delete only file1
	err = folder.DeleteObjects([]string{file1})
	assert.NoError(t, err)
	
	// base_0001 directory should still exist because file2 is still there
	dirPath := filepath.Join(tmpDir, "base_backups_005", "base_0001")
	_, err = os.Stat(dirPath)
	assert.NoError(t, err, "base_0001 directory should still exist with file2")
	
	// file2 should still exist
	file2Path := filepath.Join(tmpDir, "base_backups_005", "base_0001", "file2.txt")
	_, err = os.Stat(file2Path)
	assert.NoError(t, err, "file2 should still exist")
	
	// Now delete file2
	err = folder.DeleteObjects([]string{file2})
	assert.NoError(t, err)
	
	// Now the directory should be removed
	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err), "base_0001 directory should be removed after deleting file2")
	
	// base_backups_005 should also be removed
	dirPath = filepath.Join(tmpDir, "base_backups_005")
	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err), "base_backups_005 directory should be removed")
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
