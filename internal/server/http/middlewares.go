package http

import (
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

func FilterIP(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trustedSubnet != "" {
				_, ipNet, err := net.ParseCIDR(trustedSubnet)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to parse CIDR")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if !ipNet.Contains(net.ParseIP(r.RemoteAddr)) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
