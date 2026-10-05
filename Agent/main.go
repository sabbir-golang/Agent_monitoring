package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/kbinani/screenshot"
)

const (
	serverURL    = "http://192.168.1.71:8080"
	streamPath   = "/stream"
	framesPerSec = 5
	jpegQuality  = 60
)

var (
	cachedIP   string
	cachedHost string
	httpClient = &http.Client{Timeout: 5 * time.Second}
)

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return "unknown"
}

func Hostname(ip string) string {
	if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
func getActiveWindow() string {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	default:
		// Windows / macOS support not implemented yet.
		return ""
	}
}

func captureJPEG() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

func uploadFrame(jpegBytes []byte) error {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("ip", cachedIP)
	_ = w.WriteField("host", cachedHost)
	_ = w.WriteField("window", getActiveWindow())
	_ = w.WriteField("ts", fmt.Sprintf("%d", time.Now().UnixMilli()))

	part, err := w.CreateFormFile("frame", "frame.jpg")
	if err != nil {
		return err
	}
	if _, err := part.Write(jpegBytes); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, serverURL+streamPath, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	// Drain & close so the connection can be reused (keep-alive).
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

func main() {
	cachedIP = getIP()
	cachedHost = Hostname(cachedIP)
	log.Printf("agent starting — ip=%s host=%s os=%s fps=%d",
		cachedIP, cachedHost, runtime.GOOS, framesPerSec)

	interval := time.Second / time.Duration(framesPerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	failures := 0
	for range ticker.C {
		frame, err := captureJPEG()
		if err != nil {
			log.Printf("capture error: %v", err)
			continue
		}
		if err := uploadFrame(frame); err != nil {
			failures++
			log.Printf("upload error: %v", err)

			if failures >= 5 {
				time.Sleep(2 * time.Second)
			}
			continue
		}
		failures = 0
	}
}
