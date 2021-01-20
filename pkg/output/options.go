package output

import "time"

type Options struct {
	Addr        string        // Destination address (host:port).
	Delay       time.Duration // Delay start after start signal.
	Protocol    string        // Protocol (udp/tcp/tls).
	Retries     int           // Number of connection retries for tcp based protocols.
	StartSignal string        // OS signal to wait on before starting.
}
