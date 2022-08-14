package sender

import (
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

type CustomTransport struct {
	http.RoundTripper
}

func (t CustomTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ip, err := getOutboundIP()
	if err != nil {
		return nil, err
	}
	r.Header.Set("X-Real-IP", ip.String())
	return t.RoundTripper.RoundTrip(r)
}

func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close connection")
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}
