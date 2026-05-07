package greenplum

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

// Connect establishes a connection to GreenPlum using
// a UNIX socket. Must export PGHOST and run with `sudo -E -u postgres`.
// If PGHOST is not set or if the connection fails, an error is returned
// and the connection is `<nil>`.
//
// This function tries to connect with GreenPlum-specific parameters (gp_role, gp_session_role)
// for connecting to segments in utility mode. If those fail, it falls back to standard connection.
//
// Example: PGHOST=/var/run/postgresql or PGHOST=10.0.0.1
func Connect(configOptions ...func(config *pgx.ConnConfig) error) (*pgx.Conn, error) {
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, errors.Wrap(err, "Connect: unable to read environment variables")
	}

	// apply passed custom config options, if any
	for _, option := range configOptions {
		err := option(config)
		if err != nil {
			return nil, err
		}
	}

	// Try standard connection first (without GP-specific parameters)
	conn, err := pgx.ConnectConfig(context.TODO(), config)
	if err != nil {
		// If standard connection fails, try to connect to GP segment with utility mode
		conn, err = tryConnectToGpSegment(config)

		if err != nil && config.Host != "localhost" {
			tracelog.ErrorLogger.Println(err.Error())
			tracelog.ErrorLogger.Println("Failed to connect using provided PGHOST and PGPORT, trying localhost:5432")
			config.Host = "localhost"
			config.Port = 5432
			conn, err = pgx.ConnectConfig(context.TODO(), config)
		}

		if err != nil {
			return nil, errors.Wrap(err, "Connect: greenplum connection failed")
		}
	}

	return conn, nil
}

// tryConnectToGpSegment attempts to connect to a GreenPlum segment in utility mode.
// It first tries with gp_role parameter, and if that fails, adds gp_session_role as well.
// Note: Both parameters will be set on the second attempt for maximum compatibility.
func tryConnectToGpSegment(config *pgx.ConnConfig) (*pgx.Conn, error) {
	config.RuntimeParams["gp_role"] = "utility"
	conn, err := pgx.ConnectConfig(context.TODO(), config)

	if err != nil {
		// Add gp_session_role (in addition to gp_role) for older GreenPlum versions
		config.RuntimeParams["gp_session_role"] = "utility"
		conn, err = pgx.ConnectConfig(context.TODO(), config)
	}
	return conn, err
}
