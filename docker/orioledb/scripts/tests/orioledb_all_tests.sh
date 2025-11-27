#!/bin/sh

# Combined test script that runs all OrioleDB tests sequentially
# This avoids rebuilding the OrioleDB docker image for each test

set -e -x

echo "=== Running OrioleDB Compatibility Test ==="
/tmp/tests/compatibility_test.sh

echo "=== Running OrioleDB Simple Test ==="
/tmp/tests/simple_test.sh

echo "=== Running OrioleDB Compressed Test ==="
/tmp/tests/compressed_test.sh

echo "=== All OrioleDB Tests Completed Successfully ==="
