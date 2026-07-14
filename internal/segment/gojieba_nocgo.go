//go:build !cgo

package segment

import "errors"

func newGojieba(_ string) (Segmenter, error) {
	return nil, errors.New("gojieba requires CGO")
}
