package binary

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/utility"
)

type MongodProcess struct {
	minimalConfigPath string
	parameters        []string
	replsetID         string
	isMongoCfg        bool
	restore           bool
	port              int
	cancel            context.CancelFunc
	cmd               *exec.Cmd
}

func StartMongoWithRestore(minimalConfigPath string) (*MongodProcess, error) {
	mongodProcess := &MongodProcess{
		minimalConfigPath: minimalConfigPath,
		restore:           true,
	}

	err := mongodProcess.start()
	if err != nil {
		return nil, err
	}
	return mongodProcess, nil
}

func StartMongodWithDisableLogicalSessionCacheRefresh(minimalConfigPath string) (*MongodProcess, error) {
	return StartMongo(minimalConfigPath, "", false,
		"disableLogicalSessionCacheRefresh=true", "skipShardingConfigurationChecks=true")
}

func StartMongodWithRecoverFromOplogAsStandalone(minimalConfigPath string) (*MongodProcess, error) {
	return StartMongo(minimalConfigPath, "", false,
		"recoverFromOplogAsStandalone=true", "takeUnstableCheckpointOnShutdown=true")
}

func StartMongodWithReplyOplogAsStandalone(minimalConfigPath string, replsetID string, isMongoCfg bool) (*MongodProcess, error) {
	return StartMongo(minimalConfigPath, replsetID, isMongoCfg,
		"disableLogicalSessionCacheRefresh=true", "takeUnstableCheckpointOnShutdown=true")
}

func StartMongo(minimalConfigPath string, replsetID string, isMongoCfg bool, parameters ...string) (*MongodProcess, error) {
	mongodProcess := &MongodProcess{
		minimalConfigPath: minimalConfigPath,
		parameters:        parameters,
	}

	err := mongodProcess.start()
	if err != nil {
		return nil, err
	}
	return mongodProcess, nil
}

func (mongodProcess *MongodProcess) GetHostWithPort() string {
	return fmt.Sprintf("localhost:%d", mongodProcess.port)
}

func (mongodProcess *MongodProcess) GetURI() string {
	return "mongodb://" + mongodProcess.GetHostWithPort()
}

func (mongodProcess *MongodProcess) Wait() error {
	tracelog.InfoLogger.Printf("Waiting for the mongod %v to be stopped", mongodProcess.GetURI())
	err := mongodProcess.cmd.Wait()
	if err != nil {
		tracelog.ErrorLogger.Printf("Mongod %v stopped with error %v", mongodProcess.GetURI(), err)
	} else {
		tracelog.InfoLogger.Printf("Mongod %v stopped successfully!", mongodProcess.GetURI())
	}
	return err
}

func (mongodProcess *MongodProcess) Close() {
	mongodProcess.cancel()
}

func (mongodProcess *MongodProcess) start() (err error) {
	mongodProcess.port, err = randomUnusedPort()
	if err != nil {
		return err
	}

	cliArgs := []string{"--port", strconv.Itoa(mongodProcess.port), "--config", mongodProcess.minimalConfigPath}
	for _, parameter := range mongodProcess.parameters {
		cliArgs = append(cliArgs, "--setParameter", parameter)
	}
	if len(mongodProcess.replsetID) != 0 {
		cliArgs = append(cliArgs, "--replSet", mongodProcess.replsetID)
	}

	if mongodProcess.isMongoCfg {
		cliArgs = append(cliArgs, "--configsvr")
	}
	if mongodProcess.restore {
		cliArgs = append(cliArgs, "--restore")
	}

	ctx, cancel := context.WithCancel(context.Background())
	mongodProcess.cmd = exec.CommandContext(ctx, "mongod", cliArgs...)

	tracelog.InfoLogger.Printf("Starting mongod by command: %v", mongodProcess.cmd)
	err = mongodProcess.cmd.Start()
	if err != nil {
		cancel()
		return err
	}

	mongodProcess.cancel = cancel
	return nil
}

func randomUnusedPort() (int, error) {
	listen, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, errors.Wrap(err, "unable to choose random unused port")
	}
	defer utility.LoggedClose(listen, "unable to close listen")

	port := listen.Addr().(*net.TCPAddr).Port
	return port, nil
}
