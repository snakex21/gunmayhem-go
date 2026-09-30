//go:build !windows

package game

func readClipboardText() (string, error) { return "", errClipboardUnsupported }
func writeClipboardText(string) error    { return errClipboardUnsupported }
