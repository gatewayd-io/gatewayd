package plugin

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"
)

func sha256sum(filename string) (string, error) {
	if info, err := os.Stat(filename); err != nil || info.IsDir() {
		return "", err
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hashAlgorithm := sha256.New()

	buf := make([]byte, 65536)
	for {
		switch n, err := bufio.NewReader(file).Read(buf); err {
		case nil:
			hashAlgorithm.Write(buf[:n])
		case io.EOF:
			return fmt.Sprintf("%x", hashAlgorithm.Sum(nil)), nil
		default:
			return "", err
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