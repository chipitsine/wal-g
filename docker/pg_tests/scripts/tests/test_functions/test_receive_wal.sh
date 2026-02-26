#!/bin/sh
set -e
test_receive_wal()
{
  TMP_CONFIG=$1
  initdb ${PGDATA}

  pg_ctl -D ${PGDATA} -w start

  wal-g --config=${TMP_CONFIG} wal-receive &
  WAL_RECEIVE_PID=$!

  pgbench -i -s 5 postgres
  pg_dumpall -f /tmp/dump1
  pgbench -c 2 -T 10 -S
  pg_ctl -D ${PGDATA} -w stop -m fast
  wait $WAL_RECEIVE_PID || true
  VERIFY_OUTPUT=$(mktemp)
  # Verify and store in temp file
  wal-g --config=${TMP_CONFIG} wal-verify integrity > "${VERIFY_OUTPUT}"

  # parse verify results
  VERIFY_RESULT=$(awk 'BEGIN{FS=":"}$1~/integrity check status/{print $2}' $VERIFY_OUTPUT)

  cat "${VERIFY_OUTPUT}"

  # check verify results to end with 'OK'
  if echo "$VERIFY_RESULT" | grep -qP "\bOK$"; then
    /tmp/scripts/drop_pg.sh
    rm ${TMP_CONFIG}
    echo "WAL receive success!!!!!!"
    return 0
  fi
  echo "WAL not received as expected!!!!!"
  return 1
}
