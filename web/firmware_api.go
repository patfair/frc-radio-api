package web

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"filippo.io/age"
	"fmt"
	"github.com/patfair/frc-radio-api/radio"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"time"
)

const (
	// Maximum size of the firmware file that can be uploaded.
	maxRequestSizeBytes = 64 * 1024 * 1024 // 64 MB

	// Maximum size of the firmware file that can be held in memory at once (based on device memory limitations).
	maxMemorySizeBytes = 2 * 1024 * 1024 // 2 MB

	// Path to the optional file containing the private key for decrypting new firmware.
	firmwareDecryptionKeyFilePath = "/root/frc-radio-api-firmware-key.txt"

	// Path where new firmware files are saved after being decrypted.
	firmwarePath = "/tmp/new-firmware.tar"
)

var checksumRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// firmwareHandler handles requests to update the radio firmware.
func (web *WebServer) firmwareHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		handleWebErr(
			w,
			errors.New("not authorized; must provide 'Authorization: Bearer [password]' header"),
			http.StatusUnauthorized,
		)
		return
	}

	// Prevent a malicious client from uploading a huge file and filling up the disk.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSizeBytes)
	if err := r.ParseMultipartForm(maxMemorySizeBytes); err != nil {
		handleWebErr(w, fmt.Errorf("error parsing multipart form: %v", err), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		handleWebErr(w, fmt.Errorf("missing or invalid firmware file: %v", err), http.StatusBadRequest)
		return
	}

	checksum := r.FormValue("checksum")
	if !checksumRe.MatchString(checksum) {
		handleWebErr(
			w,
			errors.New(
				"missing or invalid checksum; expecting a 64-character hexadecimal-encoded SHA-256 hash of the "+
					"decrypted firmware file",
			),
			http.StatusBadRequest,
		)
		return
	}

	if err = web.decryptAndSaveFirmwareFile(file); err != nil {
		handleWebErr(w, fmt.Errorf("error saving firmware file: %v", err), http.StatusUnprocessableEntity)
		return
	}

	// Verify the checksum of the firmware file, reading it back from disk.
	fileChecksum, err := hashFirmwareFile()
	if err != nil {
		handleWebErr(w, fmt.Errorf("error hashing firmware file: %v", err), http.StatusInternalServerError)
		return
	}
	if fileChecksum != checksum {
		handleWebErr(
			w, fmt.Errorf("checksum mismatch; expected %s, got %s", checksum, fileChecksum), http.StatusBadRequest,
		)
		return
	}

	// Initiate the firmware update process; the radio will reboot automatically after this.
	go func() {
		// Add a short delay to give the HTTP response time to be sent.
		time.Sleep(10 * time.Millisecond)
		radio.TriggerFirmwareUpdate(firmwarePath)
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintln(w, "New firmware received and will be applied now. The radio will reboot several times. The firmware upgrade process is complete when the SYS light is slowly blinking.")
}

// decryptAndSaveFirmwareFile decrypts the given uploaded file and saves it to the hardcoded path for new firmware.
func (web *WebServer) decryptAndSaveFirmwareFile(file multipart.File) error {
	// Decrypt the firmware file if a decryption key is present; otherwise pass it through unmodified.
	var decryptedFile io.Reader
	if web.firmwareDecryptionKey != nil {
		var err error
		if decryptedFile, err = age.Decrypt(file, web.firmwareDecryptionKey); err != nil {
			log.Printf("Error decrypting firmware file: %v", err)
			return errors.New("error decrypting firmware file: incorrect key or file not encrypted")
		}
	} else {
		log.Println("No firmware decryption key specified; will assume firmware file is not encrypted.")
		decryptedFile = file
	}

	// Save the decrypted file to disk.
	dst, err := os.Create(firmwarePath)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dst, decryptedFile); err != nil {
		return err
	}
	return nil
}

// hashFirmwareFile returns the SHA-256 hash of the firmware file.
func hashFirmwareFile() (string, error) {
	hash := sha256.New()
	file, err := os.Open(firmwarePath)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
