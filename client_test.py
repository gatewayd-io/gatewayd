import os
from pprint import pprint
from concurrent.futures import ThreadPoolExecutor
import psycopg


def worker(id):
    conn = None
    try:
        conn = psycopg.connect(host="localhost", port=15432, dbname="test",
                               user="postgres", password="postgres", sslmode="disable")

        conn.execute("BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE")
        conn.execute(
            "CREATE TABLE IF NOT EXISTS test (id serial PRIMARY KEY, num integer, data varchar);")
        conn.execute("INSERT INTO test (num, data) VALUES (%s, %s)", (id, "abc'def"))
        conn.execute("COMMIT")

        for row in conn.execute("SELECT * FROM test;"):
            print("ID=%s, NUM=%s, DATA=%s" % row)

        # conn.execute("DROP TABLE test;")
        conn.close()
    except KeyboardInterrupt:
        if conn:
            conn.close()
        os._exit(0)
    except Exception as e:
        print("Worker %s: %s" % (id, e))

    return


if __name__ == '__main__':
    with ThreadPoolExecutor(max_workers=10) as executor:
        # Create 11 connections to the server and run queries in parallel
        # This will cause the server to crash
        for i in range(11):
            executor.submit(worker, i)
