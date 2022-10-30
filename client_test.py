import os
from pprint import pprint
from concurrent.futures import ThreadPoolExecutor
import psycopg


def create_table():
    conn = None
    try:
        conn = psycopg.connect(host="localhost", port=15432, dbname="test",
                               user="postgres", password="postgres", sslmode="disable")

        conn.execute(
            "CREATE TABLE IF NOT EXISTS test (id serial PRIMARY KEY, num integer, data varchar);")
        conn.close()
    except KeyboardInterrupt:
        if conn:
            conn.close()
        os._exit(0)
    except Exception as e:
        print("Worker %s: %s" % (id, e))

    return


def writer(id):
    conn = None
    try:
        conn = psycopg.connect(host="localhost", port=15432, dbname="test",
                               user="postgres", password="postgres", sslmode="disable")

        conn.execute("INSERT INTO test (num, data) VALUES (%s, %s)", (id, "abc'def"))

        conn.close()
    except KeyboardInterrupt:
        if conn:
            conn.close()
        os._exit(0)
    except Exception as e:
        print("Worker %s: %s" % (id, e))

    return


def reader():
    conn = None
    try:
        conn = psycopg.connect(host="localhost", port=15432, dbname="test",
                               user="postgres", password="postgres", sslmode="disable")

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
        executor.submit(create_table)

        for i in range(10):
            executor.submit(writer, i)

        # Wait for all threads to finish
        executor.submit(reader)
        executor.shutdown(wait=True)
