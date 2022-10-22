import psycopg

conn = psycopg.connect(host="localhost", port=15432, dbname="postgres",
                       user="postgres", password="postgres", sslmode="disable")

conn.execute("CREATE TABLE IF NOT EXISTS test (id serial PRIMARY KEY, num integer, data varchar);")
conn.execute("INSERT INTO test (num, data) VALUES (%s, %s)", (100, "abc'def"))

for row in conn.execute("SELECT * FROM test;"):
    print("ID=%s, NUM=%s, DATA=%s" % row)

conn.execute("DROP TABLE test;")

conn.close()
