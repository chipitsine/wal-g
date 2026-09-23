package internal

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wal-g/tracelog"
	conf "github.com/wal-g/wal-g/internal/config"
	"github.com/wal-g/wal-g/utility"
)

type FileTarInterpreter struct {
	DirectoryToSave string
	fsync           bool
}

func NewFileTarInterpreter(directoryToSave string) TarInterpreter {
	return &FileTarInterpreter{
		DirectoryToSave: directoryToSave,
		fsync:           !viper.GetBool(conf.TarDisableFsyncSetting),
	}
}

func (tarInterpreter *FileTarInterpreter) Interpret(reader io.Reader, fileInfo *tar.Header) error {
	tracelog.DebugLogger.Println("Interpreting: ", fileInfo.Name)
	targetPath := path.Join(tarInterpreter.DirectoryToSave, fileInfo.Name)
	switch fileInfo.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		return tarInterpreter.interpretRegularFile(targetPath, fileInfo, reader)
	case tar.TypeDir:
		err := os.MkdirAll(targetPath, 0755)
		if err != nil {
			return errors.Wrapf(err, "Interpret: failed to create all directories in %s", targetPath)
		}
		if err = os.Chmod(targetPath, os.FileMode(fileInfo.Mode)); err != nil {
			return errors.Wrap(err, "Interpret: chmod failed")
		}
	case tar.TypeLink:
		if err := CreateHardLinkOrCopy(fileInfo.Name, targetPath); err != nil {
			return errors.Wrapf(err, "Interpret: failed to create hardlink %s", targetPath)
		}
	case tar.TypeSymlink:
		if err := os.Symlink(fileInfo.Linkname, targetPath); err != nil {
			return errors.Wrapf(err, "Interpret: failed to create symlink %s", targetPath)
		}
	}
	return nil
}

func (tarInterpreter *FileTarInterpreter) interpretRegularFile(targetPath string, header *tar.Header, reader io.Reader) error {
	localFile, _, err := utility.GetLocalFile(targetPath, header)
	if err != nil {
		return err
	}
	defer utility.LoggedClose(localFile, "")
	defer utility.LoggedSync(localFile, "", tarInterpreter.fsync)

	return utility.WriteLocalFile(reader, header, localFile, tarInterpreter.fsync)
}

func CreateHardLinkOrCopy(sourcePath, targetPath string) error {
	err := os.Link(sourcePath, targetPath)
	if err == nil {
		return nil
	}
	if !IsCrossVolumeHardLinkError(err) {
		return err
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer utility.LoggedClose(sourceFile, "")

	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer utility.LoggedClose(targetFile, "")

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return err
	}
	return targetFile.Sync()
}

func IsCrossVolumeHardLinkError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "exdev") ||
		strings.Contains(msg, "cross-device") ||
		strings.Contains(msg, "different volume") ||
		strings.Contains(msg, "not same device") ||
		strings.Contains(msg, "not same file system")
}
