package wgtester

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fosrl/newt/logger"
)

const (
	// Magic bytes to identify our packets
	magicHeader uint32 = 0xDEADBEEF
	// Request packet type
	packetTypeRequest uint8 = 1
	// Response packet type
	packetTypeResponse uint8 = 2
	// Packet format:
	// - 4 bytes: magic header (0xDEADBEEF)
	// - 1 byte: packet type (1 = request, 2 = response)
	// - 8 bytes: timestamp (for round-trip timing)
	packetSize = 13
)

// Server handles listening for connection check requests using UDP
type Server struct {
	conn         *net.UDPConn
	serverAddr   string
	serverPort   uint16
	shutdownCh   chan struct{}
	isRunning    bool
	runningLock  sync.Mutex
	newtID       string
	outputPrefix string
}

// NewServer creates a new connection test server using UDP
func NewServer(serverAddr string, serverPort uint16, newtID string) *Server {
	return &Server{
		serverAddr:   serverAddr,
		serverPort:   serverPort + 1, // use the next port for the server
		shutdownCh:   make(chan struct{}),
		newtID:       newtID,
		outputPrefix: "[WGTester] ",
	}
}

// Start begins listening for connection test packets using UDP
func (s *Server) Start() error {
	s.runningLock.Lock()
	defer s.runningLock.Unlock()

	if s.isRunning {
		return nil
	}

	//create the address to listen on
	addr := net.JoinHostPort(s.serverAddr, fmt.Sprintf("%d", s.serverPort))

	// Create UDP address to listen on
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	s.conn = conn

	s.isRunning = true
	go s.handleConnections()

	logger.Info("%sServer started on %s:%d", s.outputPrefix, s.serverAddr, s.serverPort)
	return nil
}

// Stop shuts down the server
func (s *Server) Stop() {
	s.runningLock.Lock()
	defer s.runningLock.Unlock()

	if !s.isRunning {
		return
	}

	close(s.shutdownCh)
	if s.conn != nil {
		s.conn.Close()
	}
	s.isRunning = false
	logger.Info(s.outputPrefix + "Server stopped")
}

// handleConnections processes incoming packets
func (s *Server) handleConnections() {
	buffer := make([]byte, 2000) // Buffer large enough for any UDP packet

	for {
		select {
		case <-s.shutdownCh:
			return
		default:
			// Set read deadline to avoid blocking forever
			err := s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			if err != nil {
				logger.Error(s.outputPrefix+"Error setting read deadline: %v", err)
				continue
			}

			// Read from UDP connection
			n, addr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Just a timeout, keep going
					continue
				}
				// Check if we're shutting down and the connection was closed
				select {
				case <-s.shutdownCh:
					return // Don't log error if we're shutting down
				default:
					logger.Error(s.outputPrefix+"Error reading from UDP: %v", err)
				}
				continue
			}

			// Process packet only if it meets minimum size requirements
			if n < packetSize {
				continue // Too small to be our packet
			}

			// Check magic header
			magic := binary.BigEndian.Uint32(buffer[0:4])
			if magic != magicHeader {
				continue // Not our packet
			}

			// Check packet type
			packetType := buffer[4]
			if packetType != packetTypeRequest {
				continue // Not a request packet
			}

			// Create response packet
			responsePacket := make([]byte, packetSize)
			// Copy the same magic header
			binary.BigEndian.PutUint32(responsePacket[0:4], magicHeader)
			// Change the packet type to response
			responsePacket[4] = packetTypeResponse
			// Copy the timestamp (for RTT calculation)
			copy(responsePacket[5:13], buffer[5:13])

			// Log response being sent for debugging
			logger.Debug(s.outputPrefix+"Sending response to %s", addr.String())

			// Send the response packet directly to the source address
			_, err = s.conn.WriteToUDP(responsePacket, addr)
			if err != nil {
				logger.Error(s.outputPrefix+"Error sending response: %v", err)
			} else {
				logger.Debug(s.outputPrefix + "Response sent successfully")
			}
		}
	}
}
