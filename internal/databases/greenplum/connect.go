package greenplum

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

// Connect establishes a connection to Greenplum using
// a UNIX socket. Must export PGHOST and run with `sudo -E -u postgres`.
// If PGHOST is not set or if the connection fails, an error is returned
// and the connection is `<nil>`.
//
// Example: PGHOST=/var/run/postgresql or PGHOST=10.0.0.1
//
// This function implements Greenplum-specific connection logic,
// including fallback with gp_role and gp_session_role parameters
// when the initial connection fails.
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

	conn, err := pgx.ConnectConfig(context.TODO(), config)
	if err != nil {
		conn, err = tryConnectToGpSegment(config)

		if err != nil && config.Host != "localhost" {
			tracelog.ErrorLogger.Println(err.Error())
			tracelog.ErrorLogger.Println("Failed to connect using provided PGHOST and PGPORT, trying localhost:5432")
			config.Host = "localhost"
			config.Port = 5432
			// Try localhost with GP parameters
			conn, err = tryConnectToGpSegment(config)
		}

		if err != nil {
			return nil, errors.Wrap(err, "Connect: greenplum connection failed")
		}
	}

	return conn, nil
}

// nolint:gocritic
func tryConnectToGpSegment(config *pgx.ConnConfig) (*pgx.Conn, error) {
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["gp_role"] = "utility"
	conn, err := pgx.ConnectConfig(context.TODO(), config)

	if err != nil {
		config.RuntimeParams["gp_session_role"] = "utility"
		conn, err = pgx.ConnectConfig(context.TODO(), config)
	}
	return conn, err
}
