package capture

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/log"
)

type PayloadCapture struct {
	mu         sync.Mutex
	captured   map[string]bool
	outputPath string
	maxPackets int
	count      int
}

func NewPayloadCapture(outputPath string, maxPackets int) *PayloadCapture {
	if outputPath == "" {
		outputPath = "./captures"
	}
	os.MkdirAll(outputPath, 0755)

	return &PayloadCapture{
		captured:   make(map[string]bool),
		outputPath: outputPath,
		maxPackets: maxPackets,
	}
}

func (pc *PayloadCapture) ShouldCapture() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.count < pc.maxPackets
}

func (pc *PayloadCapture) CapturePacket(protocol, domain string, payload []byte) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.count >= pc.maxPackets {
		return nil
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.bin", protocol, domain, timestamp)
	filepath := filepath.Join(pc.outputPath, filename)

	// Save binary payload
	if err := os.WriteFile(filepath, payload, 0644); err != nil {
		return err
	}

	// Also save hex version for easy config use
	hexFile := filepath + ".hex"
	if err := os.WriteFile(hexFile, []byte(hex.EncodeToString(payload)), 0644); err != nil {
		return err
	}

	// Save metadata
	metaFile := filepath + ".txt"
	metadata := fmt.Sprintf("Protocol: %s\nDomain: %s\nTimestamp: %s\nLength: %d bytes\n",
		protocol, domain, timestamp, len(payload))
	os.WriteFile(metaFile, []byte(metadata), 0644)

	pc.count++
	log.Infof("✓ Captured %s payload for %s → %s (%d/%d)",
		protocol, domain, filepath, pc.count, pc.maxPackets)

	if pc.count >= pc.maxPackets {
		log.Infof("Capture complete! Files saved in %s", pc.outputPath)
	}

	return nil
}
