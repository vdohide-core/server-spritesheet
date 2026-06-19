package models

import "fmt"

// GetHostPort returns "host:port" for internal HTTP access (storage static server).
func (s *Storage) GetHostPort() string {
	host := s.GetHost()
	if host == "" {
		return ""
	}
	port := s.GetPort()
	if port > 0 {
		return fmt.Sprintf("%s:%d", host, port)
	}
	return host + ":8888"
}
