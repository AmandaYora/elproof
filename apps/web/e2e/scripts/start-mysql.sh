#!/bin/sh
# Starts a throwaway MySQL 8 on 127.0.0.1:${E2E_MYSQL_PORT:-3399} for the e2e stack.
# Its data lives in e2e/.tmp/mysql (gitignored). It never touches any other MySQL.
set -eu
HERE="$(cd "$(dirname "$0")/.." && pwd)"
BASE="${E2E_MYSQL_BASEDIR:-/opt/homebrew/opt/mysql@8.0}"
PORT="${E2E_MYSQL_PORT:-3399}"
DATA="${E2E_MYSQL_DATADIR:-$HERE/.tmp/mysql}"
SOCK="/tmp/elproof-e2e-$PORT.sock"
mkdir -p "$HERE/.tmp"
if "$BASE/bin/mysqladmin" --no-defaults -h127.0.0.1 -P"$PORT" -uroot ping >/dev/null 2>&1; then
  echo "mysql already up on $PORT"; exit 0
fi
if [ ! -d "$DATA/mysql" ]; then
  "$BASE/bin/mysqld" --no-defaults --initialize-insecure --basedir="$BASE" --datadir="$DATA" >"$HERE/.tmp/mysql-init.log" 2>&1
fi
nohup "$BASE/bin/mysqld" --no-defaults --basedir="$BASE" --datadir="$DATA" --port="$PORT" --socket="$SOCK" \
  --mysqlx=OFF --bind-address=127.0.0.1 --pid-file="$DATA/mysqld.pid" --log-error="$HERE/.tmp/mysqld.err" \
  >"$HERE/.tmp/mysqld.out" 2>&1 &
i=0
until "$BASE/bin/mysqladmin" --no-defaults -h127.0.0.1 -P"$PORT" -uroot ping >/dev/null 2>&1; do
  i=$((i+1)); [ $i -gt 60 ] && { echo "mysql did not start"; tail -5 "$HERE/.tmp/mysqld.err"; exit 1; }
  sleep 1
done
echo "mysql up on $PORT"
