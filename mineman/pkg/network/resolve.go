package network

import (
	"errors"
	"net"
)

var (
	ErrTargetResolutionFailed = errors.New("target ip resolution is failed")
)

// ResolveIP resolve the target IP by using either ip formatted target or
// by domain, for now only support IPv4
func ResolveIP(target string) (net.IP, error) {
	// only support ipv4
	targetIP := net.ParseIP(target)
	if ipv4 := targetIP.To4(); ipv4 != nil {
		targetIP = ipv4
	} else {
		targetIP = nil
	}

	// if target is not specified, then try lookup ip
	if targetIP == nil || !targetIP.IsUnspecified() {
		addrs, err := net.LookupIP(target)
		if err != nil {
			return nil, err
		}

		// use the first ip
		for _, addr := range addrs {
			if ipv4 := addr.To4(); ipv4 != nil {
				targetIP = ipv4
				break
			}
		}
	}

	if targetIP == nil || targetIP.IsUnspecified() {
		return nil, ErrTargetResolutionFailed
	}

	return targetIP, nil
}
