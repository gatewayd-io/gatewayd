package plugin

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"
)

const bufferSize = 65536

// sha256sum returns the sha256 checksum of a file.
// Ref: https://github.com/codingsince1985/checksum
// A little copying is better than a little dependency.
func sha256sum(filename string) (string, *gerr.GatewayDError) {
	if info, err := os.Stat(filename); err != nil || info.IsDir() {
		return "", gerr.ErrFileNotFound.Wrap(err)
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", gerr.ErrFileOpenFailed.Wrap(err)
	}
	defer func() { _ = file.Close() }()

	hashAlgorithm := sha256.New()

	buf := make([]byte, bufferSize)
	for {
		n, err := bufio.NewReader(file).Read(buf)
		//nolint:gocritic
		if err == nil {
			hashAlgorithm.Write(buf[:n])
		} else if errors.Is(err, io.EOF) {
			return fmt.Sprintf("%x", hashAlgorithm.Sum(nil)), nil
		} else {
			return "", gerr.ErrFileReadFailed.Wrap(err)
		}
	}
}

// Verify compares two structs and returns true if they are equal.
func Verify(params, returnVal *structpb.Struct) bool {
	return cmp.Equal(params.AsMap(), returnVal.AsMap(), cmp.Options{
		cmpopts.SortMaps(func(a, b string) bool {
			return a < b
		}),
		cmpopts.EquateEmpty(),
	})
}

// NewCommand returns a command with the given arguments and environment variables.
func NewCommand(cmd string, args []string, env []string) *exec.Cmd {
	command := exec.Command(cmd, args...)
	command.Env = os.Environ()
	if env != nil {
		command.Env = append(command.Env, env...)
	}
	return command
}
