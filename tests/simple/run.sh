#!/bin/bash

set -e

CUR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
source $CUR/../_utils/test_prepare
WORK_DIR=$OUT_DIR/$TEST_NAME

function prepare() {
    rm -rf $WORK_DIR && mkdir -p $WORK_DIR

    start_tidb_cluster $WORK_DIR

    cd $WORK_DIR

    # record tso before we create tables for two reasons
    # 1. skip the system table DDL
    # 2. currently we support providing table IDs only when we create a changefeed, so we have to create tables before creating a changefeed.
    start_ts=$(($(date +%s%N | cut -b1-13)<<18))

    run_sql "CREATE table test.simple1(id int primary key, val int);"
    run_sql "CREATE table test.simple2(id int primary key, val int);"

    cdc server --log-file $WORK_DIR/cdc.log --log-level info > $WORK_DIR/stdout.log 2>&1 &
    cdc cli --start-ts=$start_ts
}

function sql_check() {
    # run check in sequence and short circuit principle, if error hanppens,
    # the following statement will be not executed

    # check table simple1.
    run_sql "SELECT id, val FROM test.simple1;" ${DOWN_TIDB_HOST} ${DOWN_TIDB_PORT} && \
    check_contains "id: 1" && \
    check_contains "val: 1" && \
    check_contains "id: 2" && \
    check_contains "val: 22" && \
    check_not_contains "id: 3" && \

    # check table simple2.
    run_sql "SELECT id, val FROM test.simple2;" ${DOWN_TIDB_HOST} ${DOWN_TIDB_PORT} && \
    check_contains "id: 1" && \
    check_contains "val: 1" && \
    check_contains "id: 2" && \
    check_contains "val: 22" && \
    check_not_contains "id: 3"
}

function sql_test() {
    # test insert/update/delete for two table in the same way.
    run_sql "INSERT INTO test.simple1(id, val) VALUES (1, 1);"
    run_sql "INSERT INTO test.simple1(id, val) VALUES (2, 2);"
    run_sql "INSERT INTO test.simple1(id, val) VALUES (3, 3);"

    # update id = 2 and delete id = 3
    run_sql "UPDATE test.simple1 set val = 22 where id = 2;"
    run_sql "DELETE from test.simple1 where id = 3;"


    # same dml for table simple2
    run_sql "INSERT INTO test.simple2(id, val) VALUES (1, 1);"
    run_sql "INSERT INTO test.simple2(id, val) VALUES (2, 2);"
    run_sql "INSERT INTO test.simple2(id, val) VALUES (3, 3);"

    run_sql "UPDATE test.simple2 set val = 22 where id = 2;"
    run_sql "DELETE from test.simple2 where id = 3;"

    i=0
    check_time=50
    set +e
    while [ $i -lt $check_time ]
    do
        sql_check
        ret=$?
        if [ "$ret" == 0 ]; then
            echo "check data successfully"
            break
        fi
        ((i++))
        echo "check data failed $i-th time, retry later"
        sleep 2
    done
    set -e

    if [ $i -ge $check_time ]; then
        echo "check data failed at last"
        exit 1
    fi

    killall cdc || true
}

trap stop_tidb_cluster EXIT
prepare $*
sql_test $*
echo "[$(date)] <<<<<< run test case $TEST_NAME success! >>>>>>"