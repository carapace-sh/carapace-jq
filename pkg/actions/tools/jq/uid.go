package jq

import (
	"fmt"
	"net/url"

	"github.com/carapace-sh/carapace/pkg/uid"
)

// Uid generates a UID function for carapace action deduplication.
func Uid(host string, opts ...string) func(s string, uc uid.Context) (*url.URL, error) {
	return func(s string, uc uid.Context) (*url.URL, error) {
		if length := len(opts); length%2 != 0 {
			return nil, fmt.Errorf("invalid amount of arguments [jq.Uid]: %v", length)
		}

		u := &url.URL{
			Scheme: "jq",
			Host:   host,
			Path:   s,
		}
		values := u.Query()
		for i := 0; i < len(opts); i += 2 {
			if opts[i+1] != "" {
				values.Add(opts[i], opts[i+1])
			}
		}
		u.RawQuery = values.Encode()

		return u, nil
	}
}
