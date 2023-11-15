FROM postgres:16

RUN apt-get update && apt-get install -y --no-install-recommends apt-utils
RUN apt-get -y install postgresql-16-cron

# Initialization of the postgres database, see 'Initialization scripts' from https://hub.docker.com/_/postgres
COPY 001_pg_cron.sh 002_pg_cron.sql /docker-entrypoint-initdb.d