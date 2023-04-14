package atproxy

import "net/http"

func (Server) HTTPTransports(
	dialers Dialers,
) (transports []*http.Transport) {
	for _, dial := range dialers {
		transports = append(transports, &http.Transport{
			DialContext: dial.DialContext,
		})
	}
	return
}
