package network

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/rs/zerolog"
	"github.com/xdg-go/scram"
)

// pgStartup performs the PostgreSQL startup handshake on a raw TCP connection.
// It sends a StartupMessage with the given user/database, then responds to
// whatever authentication method the server demands (trust, cleartext, MD5,
// or SCRAM-SHA-256). On success the connection is in ReadyForQuery state.
func pgStartup(conn net.Conn, user, database, password string, logger zerolog.Logger) error {
	frontend := pgproto3.NewFrontend(conn, conn)

	// Build and send the StartupMessage.
	startupMsg := &pgproto3.StartupMessage{
		ProtocolVersion: pgproto3.ProtocolVersionNumber,
		Parameters: map[string]string{
			"user":             user,
			"database":         database,
			"application_name": "gatewayd",
			"client_encoding":  "UTF8",
		},
	}
	buf, err := startupMsg.Encode(nil)
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("encode startup: %w", err))
	}
	if _, err := conn.Write(buf); err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("send startup: %w", err))
	}

	logger.Debug().Str("user", user).Str("database", database).Msg("Sent PostgreSQL StartupMessage")

	// Read messages from the server until we hit ReadyForQuery or an error.
	for {
		msg, err := frontend.Receive()
		if err != nil {
			return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("receive: %w", err))
		}

		switch msg := msg.(type) {
		case *pgproto3.AuthenticationOk:
			logger.Debug().Msg("Backend authentication successful (AuthenticationOk)")
			// Continue reading until ReadyForQuery.
		case *pgproto3.AuthenticationCleartextPassword:
			if err := pgSendPassword(conn, password); err != nil {
				return err
			}
		case *pgproto3.AuthenticationMD5Password:
			hash := pgMD5Password(user, password, msg.Salt)
			if err := pgSendPassword(conn, hash); err != nil {
				return err
			}
		case *pgproto3.AuthenticationSASL:
			if err := pgHandleSCRAM(conn, frontend, user, password, msg.AuthMechanisms, logger); err != nil {
				return err
			}
		case *pgproto3.ParameterStatus:
			logger.Trace().Str("name", msg.Name).Str("value", msg.Value).Msg("ParameterStatus")
		case *pgproto3.BackendKeyData:
			logger.Debug().Uint32("pid", msg.ProcessID).Uint32("key", msg.SecretKey).Msg("BackendKeyData")
		case *pgproto3.ReadyForQuery:
			logger.Debug().Str("txStatus", string(msg.TxStatus)).Msg("Backend ready for queries")
			return nil
		case *pgproto3.ErrorResponse:
			return gerr.ErrPgStartupFailed.Wrap(
				fmt.Errorf("backend error: %s (code %s): %s", msg.Severity, msg.Code, msg.Message))
		case *pgproto3.NoticeResponse:
			logger.Debug().Str("severity", msg.Severity).Str("message", msg.Message).Msg("NoticeResponse")
		default:
			logger.Warn().Str("type", fmt.Sprintf("%T", msg)).Msg("Unexpected message during PG startup")
		}
	}
}

// pgSendPassword sends a PasswordMessage to the server.
func pgSendPassword(conn net.Conn, password string) error {
	msg := &pgproto3.PasswordMessage{Password: password}
	buf, err := msg.Encode(nil)
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("encode password: %w", err))
	}
	if _, err := conn.Write(buf); err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("send password: %w", err))
	}
	return nil
}

// pgMD5Password computes the PostgreSQL-style MD5 password hash.
// "md5" + md5(md5(password + user) + salt).
func pgMD5Password(user, password string, salt [4]byte) string {
	// Inner hash: md5(password + user)
	inner := md5.Sum([]byte(password + user)) //nolint:gosec
	innerHex := hex.EncodeToString(inner[:])

	// Outer hash: md5(innerHex + salt)
	outer := md5.New() //nolint:gosec
	outer.Write([]byte(innerHex))
	outer.Write(salt[:])
	return "md5" + hex.EncodeToString(outer.Sum(nil))
}

// pgHandleSCRAM performs the full SCRAM-SHA-256 handshake.
func pgHandleSCRAM(
	conn net.Conn,
	frontend *pgproto3.Frontend,
	user, password string,
	mechanisms []string,
	logger zerolog.Logger,
) error {
	// Verify SCRAM-SHA-256 is offered.
	found := false
	for _, m := range mechanisms {
		if m == "SCRAM-SHA-256" {
			found = true
			break
		}
	}
	if !found {
		return gerr.ErrPgStartupFailed.Wrap(
			fmt.Errorf("server does not support SCRAM-SHA-256, offered: %v", mechanisms))
	}

	// Create SCRAM client.
	client, err := scram.SHA256.NewClient(user, password, "")
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("scram client: %w", err))
	}
	conv := client.NewConversation()

	// Step 1: Generate client-first-message and send SASLInitialResponse.
	clientFirst, err := conv.Step("")
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("scram step1: %w", err))
	}

	saslInit := &pgproto3.SASLInitialResponse{
		AuthMechanism: "SCRAM-SHA-256",
		Data:          []byte(clientFirst),
	}
	buf, err := saslInit.Encode(nil)
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("encode sasl init: %w", err))
	}
	if _, err := conn.Write(buf); err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("send sasl init: %w", err))
	}
	logger.Debug().Msg("Sent SCRAM client-first-message")

	// Step 2: Receive AuthenticationSASLContinue (server-first-message).
	msg, err := frontend.Receive()
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("receive sasl continue: %w", err))
	}
	saslContinue, isSASLContinue := msg.(*pgproto3.AuthenticationSASLContinue)
	if !isSASLContinue {
		if errResp, isErr := msg.(*pgproto3.ErrorResponse); isErr {
			return gerr.ErrPgStartupFailed.Wrap(
				fmt.Errorf("SCRAM error: %s: %s", errResp.Severity, errResp.Message))
		}
		return gerr.ErrPgStartupFailed.Wrap(
			fmt.Errorf("expected AuthenticationSASLContinue, got %T", msg))
	}

	// Step 3: Process server-first, generate client-final-message.
	clientFinal, err := conv.Step(string(saslContinue.Data))
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("scram step2: %w", err))
	}

	saslResp := &pgproto3.SASLResponse{Data: []byte(clientFinal)}
	buf, err = saslResp.Encode(nil)
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("encode sasl response: %w", err))
	}
	if _, err := conn.Write(buf); err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("send sasl response: %w", err))
	}
	logger.Debug().Msg("Sent SCRAM client-final-message")

	// Step 4: Receive AuthenticationSASLFinal (server-final-message).
	msg, err = frontend.Receive()
	if err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("receive sasl final: %w", err))
	}
	saslFinal, ok := msg.(*pgproto3.AuthenticationSASLFinal)
	if !ok {
		if errResp, isErr := msg.(*pgproto3.ErrorResponse); isErr {
			return gerr.ErrPgStartupFailed.Wrap(
				fmt.Errorf("SCRAM error: %s: %s", errResp.Severity, errResp.Message))
		}
		return gerr.ErrPgStartupFailed.Wrap(
			fmt.Errorf("expected AuthenticationSASLFinal, got %T", msg))
	}

	// Verify server signature.
	if _, err := conv.Step(string(saslFinal.Data)); err != nil {
		return gerr.ErrPgStartupFailed.Wrap(fmt.Errorf("scram step3 (verify server): %w", err))
	}
	logger.Debug().Msg("SCRAM-SHA-256 handshake completed successfully")

	// AuthenticationOk follows the SASL final message; it will be handled
	// by the main pgStartup loop.
	return nil
}

// pgResetSession sends DISCARD ALL to the backend to reset session state
// without tearing down the TCP connection. On success the connection is
// back in ReadyForQuery/idle state and can be reused.
func pgResetSession(conn net.Conn, logger zerolog.Logger) error {
	// Encode a simple Query message: "DISCARD ALL"
	query := &pgproto3.Query{String: "DISCARD ALL"}
	buf, err := query.Encode(nil)
	if err != nil {
		return gerr.ErrPgResetSessionFailed.Wrap(fmt.Errorf("encode DISCARD ALL: %w", err))
	}
	if _, err := conn.Write(buf); err != nil {
		return gerr.ErrPgResetSessionFailed.Wrap(fmt.Errorf("send DISCARD ALL: %w", err))
	}

	// Read the response: expect CommandComplete + ReadyForQuery.
	frontend := pgproto3.NewFrontend(conn, conn)
	for {
		msg, err := frontend.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return gerr.ErrPgResetSessionFailed.Wrap(errors.New("connection closed during reset"))
			}
			return gerr.ErrPgResetSessionFailed.Wrap(fmt.Errorf("receive: %w", err))
		}

		switch msg := msg.(type) {
		case *pgproto3.CommandComplete:
			logger.Trace().Str("tag", string(msg.CommandTag)).Msg("DISCARD ALL completed")
		case *pgproto3.ReadyForQuery:
			logger.Debug().Str("txStatus", string(msg.TxStatus)).Msg("Session reset, backend ready")
			return nil
		case *pgproto3.ErrorResponse:
			return gerr.ErrPgResetSessionFailed.Wrap(
				fmt.Errorf("backend error: %s (code %s): %s", msg.Severity, msg.Code, msg.Message))
		default:
			logger.Warn().Str("type", fmt.Sprintf("%T", msg)).Msg("Unexpected message during session reset")
		}
	}
}
