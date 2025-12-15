package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/resolver"
)

type Server struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Weight  int    `json:"weight"`
	Ttl     int64  `json:"ttl"`
}

func BuildPrefix(info Server) string {
	if info.Version == "" {
		return fmt.Sprintf("/%s/", info.Name)
	}
	return fmt.Sprintf("/%s/%s/", info.Name, info.Version)
}

func BuildRegisterPath(info Server) string {
	return fmt.Sprintf("%s%s", BuildPrefix(info), info.Addr)
}

func ParseValue(data []byte) (Server, error) {
	var info Server
	err := json.Unmarshal(data, &info)
	if err != nil {
		return info, err
	}
	return info, nil
}

func SplitPath(path string) (Server, error) {
	var info Server
	split := strings.Split(path, "/")
	if len(split) == 0 {
		return info, errors.New("invalid path")
	}
	info.Addr = split[len(split)-1]
	return info, nil
}

// Exist helper function
func Exist(l []resolver.Address, addr resolver.Address) bool {
	for i := range l {
		if l[i].Addr == addr.Addr {
			return true
		}
	}
	return false
}

// Remove helper function
func Remove(s []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr.Addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}
	return nil, false
}

func BuildResolverUrl(app string) string {
	return scheme + ":///" + app
}
