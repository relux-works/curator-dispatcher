//go:build !linux

package transport

func Daemon(string) error             { return ErrUnsupported }
func Capacity(string) (string, error) { return "", ErrUnsupported }
