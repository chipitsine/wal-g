package mongo

import (
	"context"
	"time"

	"github.com/wal-g/wal-g/internal"
	conf "github.com/wal-g/wal-g/internal/config"
	"github.com/wal-g/wal-g/internal/databases/mongo/binary"
	"github.com/wal-g/wal-g/utility"
)

func HandleBinaryBackupPush(ctx context.Context, permanent, skipMetadata bool, appName string) error {
	backupName := binary.GenerateNewBackupName()

	mongodbURI, err := conf.GetRequiredSetting(conf.MongoDBUriSetting)
	if err != nil {
		return err
	}
	mongodService, err := binary.CreateMongodService(ctx, appName, mongodbURI, 10*time.Minute)
	if err != nil {
		return err
	}

	uploader, err := internal.ConfigureUploader()
	if err != nil {
		return err
	}
	uploader.ChangeDirectory(utility.BaseBackupPath + "/")

	backupService, err := binary.CreateBackupService(ctx, mongodService, uploader)
	if err != nil {
		return err
	}

	return backupService.DoBackup(backupName, permanent, skipMetadata)
}
