package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/anacrolix/dms/dlna/dms"
	"github.com/sienar-tools/pico-dlna/internal/fsutil"
)

func defaultIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	c := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func main() {
	name := flag.String("name", "Pico-DLNA", "friendly name for DLNA clients")
	port := flag.String("port", ":1338", "HTTP server port (e.g. :1338, 1338, or 0.0.0.0:1338)")
	root := flag.String("path", "", "root directory with media content (required)")
	flag.Parse()

	if *root == "" {
		fmt.Fprintf(os.Stderr, "Error: -path is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Port: if numeric only (e.g. "1338"), prepend ":"
	if *port != "" && !strings.Contains(*port, ":") {
		*port = ":" + *port
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		slog.Error("failed to resolve path", "path", *root, "error", err)
		os.Exit(1)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		slog.Error("path not accessible", "path", absRoot, "error", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		slog.Error("path is not a directory", "path", absRoot)
		os.Exit(1)
	}

	fsys, err := fsutil.NewNoSymlinkFS(absRoot)
	if err != nil {
		slog.Error("failed to create filesystem", "path", absRoot, "error", err)
		os.Exit(1)
	}

	httpConn, err := net.Listen("tcp", *port)
	if err != nil {
		slog.Error("failed to listen", "port", *port, "error", err)
		os.Exit(1)
	}

	logger := slog.Default().With(slog.String("component", "pico-dlna"))

	interfaces, err := net.Interfaces()
	if err != nil {
		slog.Error("failed to get network interfaces", "error", err)
		os.Exit(1)
	}
	var validIfs []net.Interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.MTU > 0 {
			validIfs = append(validIfs, iface)
		}
	}

	icon := defaultIcon()
	_, ipv4All, _ := net.ParseCIDR("0.0.0.0/0")
	_, ipv6All, _ := net.ParseCIDR("::/0")
	server := &dms.Server{
		Logger:         logger.With(slog.String("subcomponent", "dms")),
		Interfaces:     validIfs,
		HTTPConn:       httpConn,
		FriendlyName:   *name,
		FS:             fsys,
		NoTranscode:    true,
		NoProbe:        true,
		Icons:          []dms.Icon{{Width: 48, Height: 48, Depth: 8, Mimetype: "image/png", Bytes: icon}},
		AllowedIpNets: []*net.IPNet{ipv4All, ipv6All},
	}

	if err := server.Init(); err != nil {
		slog.Error("failed to init server", "error", err)
		os.Exit(1)
	}

	slog.Info("starting pico-dlna",
		"name", *name,
		"port", *port,
		"path", absRoot,
	)

	go func() {
		if err := server.Run(); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs

	if err := server.Close(); err != nil {
		slog.Error("error closing server", "error", err)
	}
}
