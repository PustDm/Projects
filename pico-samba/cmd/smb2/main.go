// Package main - полноценная реализация SMB2/3 на чистом Go (без Samba).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	smb2 "github.com/macos-fuse-t/go-smb2/server"
	"github.com/macos-fuse-t/go-smb2/vfs"
	"github.com/sienar/pico-samba/internal/readonlyfs"
)

func main() {
	name := flag.String("name", "pico-samba", "Share name for clients")
	port := flag.Int("port", 445, "TCP port to listen on")
	dir := flag.String("dir", "", "Root directory with content (required)")
	flag.Parse()

	if *dir == "" {
		log.Fatal("directory is required, use -dir")
	}

	rootPath, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("resolve path: %v", err)
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("directory does not exist: %s", rootPath)
		}
		log.Fatalf("stat directory: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("not a directory: %s", rootPath)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = *name
	}

	fs := readonlyfs.New(rootPath)
	srv := smb2.NewServer(
		&smb2.ServerConfig{
			AllowGuest: true,
			MaxIOReads: 32,
			MaxIOWrites: 0,
			Xatrrs:     false,
		},
		&smb2.NTLMAuthenticator{
			TargetSPN:   "",
			NbDomain:    hostname,
			NbName:      hostname,
			DnsName:     hostname + ".local",
			DnsDomain:   ".local",
			AllowGuest:  true,
			UserPassword: map[string]string{
				"guest": "", "Guest": "", "GUEST": "",
				"": "",
			},
		},
		map[string]vfs.VFSFileSystem{*name: fs},
	)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("pico-samba (pure Go): share=%s port=%d dir=%s", *name, *port, rootPath)

	go func() {
		if err := srv.Serve(addr); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down")
	srv.Shutdown()
}
