package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func main() {
	mode := flag.String("mode", "", "Specify 'server' or 'client'")
	flag.Parse()

	switch strings.ToLower(*mode) {
	case "server":
		StartServer()
	case "client":
		StartClient()
	default:
		log.Fatal("Invalid mode. Please specify 'server' or 'client'.")
	}
}

func StartServer() {
	// Generate the SSH private key and read the corresponding public key
	privateBytes, err := ioutil.ReadFile("KEYGOESHERE")
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	privateKey, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Configure the SSH server
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Allow any SSH connection
			return nil, nil
		},
	}

	config.AddHostKey(privateKey)

	// Start listening on the specified port
	listener, err := net.Listen("tcp", "localhost:9283")
	if err != nil {
		log.Fatalf("Failed to listen on port 9283: %v", err)
	}

	log.Println("SSH server is listening on port 9283")

	// Accept and handle SSH connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}

		go handleSSHConnection(conn, config)
	}
}

func StartClient() {
	// Set up the SSH client configuration
	config := &ssh.ClientConfig{
		User: "deer", // Replace with your username
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Perform host key verification here
			// You can use ssh.FixedHostKey to compare against a known host key

			// Example: Accept any host key without verification
			return nil
		},
	}

	conn, err := ssh.Dial("tcp", "127.0.0.1:9283", config)
	if err != nil {
		log.Fatal(err)
	}

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	// Set Stdout and Stderr
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Execute a command on the remote server
	cmd := "ls -al"
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	fmt.Println("Command output:")
	fmt.Println(string(output))

}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig) {
	// Perform the SSH handshake to establish the connection
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to establish SSH connection: %v", err)
		return
	}

	log.Printf("SSH connection established from %s", sshConn.RemoteAddr())

	// Discard any out-of-band requests
	go ssh.DiscardRequests(reqs)

	// Handle SSH channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept SSH channel: %v", err)
			break
		}

		// Handle SSH channel requests
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "shell":
					channel.Write([]byte("Shell request received\n"))

				default:
					req.Reply(false, nil)
				}
			}
		}(requests)
	}
}
