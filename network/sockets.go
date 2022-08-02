package network

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	wire "github.com/jeroenrinzema/psql-wire"
	"github.com/lib/pq/oid"
	pg_query "github.com/pganalyze/pg_query_go"
)

type ListenerCfg struct {
	Protocol    string
	Address     string
	ConnHandler func(net.Conn, *DialerCfg)
	DialerCfg   *DialerCfg
}

type DialerCfg struct {
	ZeroCopy bool
	Protocol string
	Address  string
}

func NewListenerCfg(cfg *ListenerCfg) *ListenerCfg {
	if cfg.Protocol == "" {
		cfg.Protocol = "tcp"
	}

	if cfg.Address == "" {
		cfg.Address = ":15432"
	}

	if cfg.ConnHandler == nil {
		cfg.ConnHandler = ProxyHandler
	}

	return cfg
}

func NewDialerCfg(cfg *DialerCfg) *DialerCfg {
	if cfg.Protocol == "" {
		cfg.Protocol = "tcp"
	}

	if cfg.Address == "" {
		cfg.Address = ":5432"
	}

	return cfg
}

func NewListener(listenerCfg *ListenerCfg) error {
	// // Listen for incoming connections.
	// listener, err := net.Listen(listenerCfg.Protocol, listenerCfg.Address)
	// if err != nil {
	// 	panic(err)
	// }
	// defer listener.Close()
	// host, port, err := net.SplitHostPort(listener.Addr().String())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Listening on host: %s, port: %s\n", host, port)

	// for {
	// 	// Listen for an incoming conn
	// 	conn, err := listener.Accept()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	// Handle connections in a new goroutine
	// 	go listenerCfg.ConnHandler(conn, nil)
	// }

	wire.ListenAndServe(
		listenerCfg.Address,
		func(ctx context.Context, query string, writer wire.DataWriter) error {
			// Parse the query.
			statements, err := parser.Parse(query)
			if err != nil {
				return err
			}
			for _, stmt := range statements {
				fmt.Printf("postgresql-parser: %#v\n", stmt)
			}

			// // Parse the query into a ParseTreeList.
			// parsetreelist, err := pg_query.Parse(query)
			// if err != nil {
			// 	return err
			// }
			// fmt.Printf("pg_query: %#v\n", parsetreelist)

			// // Parse the query and create a fingerprint.
			// fingerprint, err := pg_query.FastFingerprint(query)
			// if err != nil {
			// 	return err
			// }
			// fmt.Printf("pg_query fingerprint: %s\n", fingerprint)

			// Parse the query into a JSON string.
			tree, err := pg_query.ParseToJSON(query)
			if err != nil {
				return err
			}
			fmt.Printf("pg_query: %s\n", tree)

			table := wire.Columns{
				{
					Table:  0,
					Name:   "id",
					Width:  4,
					Oid:    oid.T_int4,
					Format: wire.TextFormat,
				},
				{
					Table:  0,
					Name:   "name",
					Width:  32,
					Oid:    oid.T_text,
					Format: wire.TextFormat,
				},
			}
}

func NewDialer(dialerCfg *DialerCfg) (net.Conn, error) {
	// Dial the connection.
	conn, err := net.Dial(dialerCfg.Protocol, dialerCfg.Address)
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	fmt.Printf("Connected to host: %s, port: %s\n", host, port)

	return conn, nil
}

func ProxyHandler(src net.Conn, dialerCfg *DialerCfg) {
	// Create a new proxy
	if dialerCfg == nil {
		dialerCfg = NewDialerCfg(&DialerCfg{})
	}

	// Create a new connection to the database
	dst, err := NewDialer(dialerCfg)
	if err != nil {
		fmt.Printf("Error dialing: %#v\n", err)
		// Failed to connect to the database, close the connection
		// TODO: this should happen gracefully
		src.Close()
		return
	}

	if dialerCfg.ZeroCopy {
		// Zero-copy proxy
		// This causes the proxy to happen in the kernel, which is faster
		// than copying data between the two connections, but we can't
		// change the data.
		// TODO: might remove eventually
		go func() {
			io.Copy(dst, src)
			dst.Close()
			src.Close()
		}()
		go func() {
			io.Copy(src, dst)
			dst.Close()
			src.Close()
		}()
	} else {
		// Copy data from source to destination and vice versa
		// This is slower than zero-copy, but we can change the data,
		// parse it, optimize it, etc.
		go func() {
			var buf []byte
			for {
				buf = make([]byte, 1024)
				size, err := src.Read(buf)
				if err != nil {
					fmt.Printf("Error reading: %#v\n", err)
					break
				}

				binary_query := buf[:size]
				query := getQuery(binary_query)

				fmt.Printf("C2S: %s\n", query)
				fmt.Printf("Parsed Query: %s\n", parseQuery(query))

				dst.Write(binary_query)
			}
			dst.Close()
			src.Close()
		}()
		go func() {
			var buf []byte
			for {
				buf = make([]byte, 1024)
				len, err := dst.Read(buf)
				if err != nil {
					fmt.Printf("Error reading: %#v\n", err)
					break
				}
				fmt.Printf("S2C: %s %s\n", string(buf[0]), string(buf[1:len]))
				src.Write(buf[:len])
			}
			dst.Close()
			src.Close()
		}()
	}
}

func getQuery(buffer []byte) string {
	pos := bytes.IndexByte(buffer, 0)
	if pos == -1 {
		panic("Invalid query")
	}

	return string(buffer[:pos])
}

func parseQuery(query string) string {
	tree, err := pg_query.ParseToJSON(query)
	if err != nil {
		panic(err)
	}
	return tree
}
