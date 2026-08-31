//go:build !windows

package networksetup

func HandleSpecialMode([]string) (bool, error) { return false, nil }
func EnsureConfigured()                        {}
