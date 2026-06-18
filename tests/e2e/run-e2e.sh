#!/bin/bash
# SC2059: to send terminal control codes (colours) variables MUST NOT be escaped: put in format not parameters
# shellcheck disable=SC2059

set -uo pipefail

RED="\033[1;31m"
BLUE="\033[0;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

DB_NAME="arrower_test"
DB_USER="arrower"
DCF="tests/e2e/docker-compose.test.yaml"




start_services() {
    printf "${YELLOW}Starting Docker services...${RESET}\n"

    mkdir -p tmp

    # --force-recreate:     stops and removes existing containers
    # --renew-anon-volumes: anonymous volumes instead of retrieving data from the previous containers
    # => consistent & clean environment for reliable results
    docker compose -f $DCF up --force-recreate --renew-anon-volumes -d

    until docker compose -f $DCF exec -T postgres pg_isready -U "$DB_USER" >/dev/null 2>&1; do
      printf "${BLUE}  waiting for postgres...${RESET}\n"
      sleep 1
    done
}

stop_services() {
    printf "${YELLOW}Stopping services...${RESET}\n"

    # match the server: both the `go run` parent and the compiled child carry this --config arg
    pkill -9 -f 'tests/e2e/test.config.yaml' 2>/dev/null || true

    # -v: remove volumes to not keep any data around
    docker compose -f $DCF down -v || true

    printf "${GREEN}Services stopped${RESET}\n"
}

clean_between_runs() {
    printf "${YELLOW}Cleaning ...${RESET}\n"

    # match the server: both the `go run` parent and the compiled child carry this --config arg
    pkill -9 -f 'tests/e2e/test.config.yaml' 2>/dev/null || true

    # Note: Postgres prevents dropping databases with active connections.
    #       Multiple comma separated commands in psql are wrapped into a transaction.
    #       Dropping a database can not be in tx => multiple commands
    # Terminate all connections to test database
    docker compose -f $DCF exec -T postgres psql -U "$DB_USER" -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME';" > /dev/null 2>&1
    # Remove the entire database (no schema, sequence, extensions etc. survives)
    docker compose -f $DCF exec -T postgres psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;"
    # Create fresh empty database
    docker compose -f $DCF exec -T postgres psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;" > /dev/null 2>&1
}

run_tests() {
    clean_between_runs

    printf "${YELLOW}Starting Go server...${RESET}\n"
    nohup go run -tags=e2e ./tests/e2e/app --config=tests/e2e/test.config.yaml > ./tmp/e2e-server-logs.txt 2>&1 &
    disown  # remove from job control => background process does not print information on termination

    until curl --output /dev/null --silent --fail http://127.0.0.1:2223/status; do
        printf "${BLUE}  wait until Go server started...${RESET}\n"
        sleep 2
    done


    printf "${YELLOW}Run tests...${RESET}\n"

    path="./contexts/auth/tests/... ./contexts/admin/tests/..."

    if go test -tags=e2e -v $path -count=1; then
        printf "${GREEN}All tests passed${RESET}\n"
    else
        printf "${RED}Tests failed - inspect issues at:${RESET}\n"
        printf "${RED}  http://localhost:8080/auth/login${RESET}\n"
        printf "${RED}  cat tmp/e2e-server-logs.txt${RESET}\n"
    fi
}




#
# Start & run this script
#

# Trap for cleanup
trap 'stop_services; exit' INT TERM

# Run E2E tests
start_services
run_tests

while true; do
    printf "\n${YELLOW}=== Menu ===${RESET}\n"

    printf "${BLUE}r${RESET} - Rerun tests | ${BLUE}e${RESET} - Exit\n"


    printf "${YELLOW}Press key: ${RESET}"

    # read single character without Enter using built-in read
    old_stty_cfg=$(stty -g)
    stty raw -echo
    choice=$(dd if=/dev/stdin bs=1 count=1 2>/dev/null)
    stty "$old_stty_cfg"
    printf "$choice\n"

    # manual check for Ctrl+C (ASCII 3), as terminal was in raw mode
    if [[ "$(printf "%d" "'$choice")" -eq 3 ]]; then
        stop_services
        exit 130  # Standard exit code for Ctrl+C
    fi

    case "$choice" in
        r|R)
            run_tests
            ;;
        e|E)
            stop_services
            exit 0
            ;;
        *)
            printf "${RED}Invalid choice${RESET}\n"
            ;;
    esac
done