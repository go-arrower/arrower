#!/bin/sh
set -e

cat <<EOT >> "${PGDATA}"/postgresql.conf
shared_preload_libraries='pg_cron'
cron.database_name='postgres'
cron.timezone = '${PGTZ}'
EOT

# required to load pg_cron
pg_ctl restart