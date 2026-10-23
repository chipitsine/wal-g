#!/bin/bash
set -e -x

pushd /tmp

# PG_TEST_SHARD="<k>/<n>" (1 <= k <= n) runs every n-th test starting at the
# k-th, so n CI jobs can split the suite. Empty runs everything.
shard_index=0
shard_count=1
if [ -n "${PG_TEST_SHARD:-}" ]; then
  shard_index=$(( ${PG_TEST_SHARD%/*} - 1 ))
  shard_count=${PG_TEST_SHARD#*/}
  if [ "$shard_index" -lt 0 ] || [ "$shard_index" -ge "$shard_count" ]; then
    echo "invalid PG_TEST_SHARD=${PG_TEST_SHARD}, expected <k>/<n> with 1 <= k <= n"
    exit 1
  fi
fi
test_number=0

for i in tests/*.sh; do
  # Exclude pgbackrest because it needs to be run in separate container. It should be run individually.
  if [ "$i" = "tests/pgbackrest_backup_fetch_test.sh" ]; then continue ;fi
  # Exclude ssh test because of ssh server container dependency.
  if [ "$i" = "tests/ssh_backup_test.sh" ]; then continue ;fi
  # Exclude throttling test because of different s3 container for throttling.
  if [ "$i" = "tests/wal_perftest_with_throttling.sh" ]; then continue ;fi
  test_number=$((test_number + 1))
  if [ $(( (test_number - 1) % shard_count )) -ne "$shard_index" ]; then continue ;fi
  echo
  echo "===== RUNNING $i ====="
  set -x

  # A test that does not apply to this PostgreSQL version exits with 77, see
  # pg_compat.sh.
  start_time=$(date +%s)
  set +e
  ./"$i"
  status=$?
  set -e
  elapsed_seconds=$(($(date +%s) - start_time))

  set +x
  echo "===== TIMING $i elapsed_seconds=$elapsed_seconds ====="
  case $status in
    0)  echo "===== SUCCESS $i =====" ;;
    77) echo "===== SKIPPED $i =====" ;;
    *)  echo "===== FAILED  $i (exit $status) ====="; exit $status ;;
  esac
  echo

done
popd
