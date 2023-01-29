package plugin

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"
)

const bufferSize = 65536

func sha256sum(filename string) (string, error) {
	if info, err := os.Stat(filename); err != nil || info.IsDir() {
		return "", err //nolint:wrapcheck
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err //nolint:wrapcheck
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
			return "", err //nolint:wrapcheck
		}
	}
}

func Verify(params, returnVal *structpb.Struct) bool {
	return cmp.Equal(params.AsMap(), returnVal.AsMap(), cmp.Options{
		cmpopts.SortMaps(func(a, b string) bool {
			return a < b
		}),
		cmpopts.EquateEmpty(),
	})
}
