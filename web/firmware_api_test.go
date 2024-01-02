package web

import (
	"encoding/base64"
	"filippo.io/age"
	"github.com/patfair/frc-radio-api/radio"
	"github.com/stretchr/testify/assert"
	"testing"
)

const encryptedBase64 = "YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSB4V0dLRzJWeHl2YSs2enpteCtxcjRiaEMrRWhaWGNOUW9pY01w" +
	"cHNyT21RCnFmaFFld09LaUNoa0hmdVRVNWpSd3psaE5UdXo1NmxMNUdnSGlBKzlIZ3MKLS0tIDZnaitJQStHeTlySlFFS1M3VGFQYi91NzEyOU04" +
	"UE8xRGh3QlhKa05HaTAKkImLt8n/HK5tNDObg/rBSkniuquU0M/1zfor20Rbx0svTIbqgWZ06lmt2H4HSGOdn+EJsWGmNOGccj5Cig=="

func TestWeb_firmwareHandler(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)

	// Decryption not enabled.
	recorder := web.postFileHttpResponse(
		"/firmware",
		"file",
		[]byte("unencrypted firmware content\n"),
		map[string]string{"checksum": "77f2b19e93301391ed20a400a8bdb97185054b83e65ccc35c63f0895cbc59713"},
	)
	assert.Equal(t, 202, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "received and will be applied now")

	// Decryption enabled.
	web.firmwareDecryptionKey, _ = age.ParseX25519Identity(
		"AGE-SECRET-KEY-1QS7DUT0EK9LYHRXYJLLFDM26MALP78UTT48TNPZS55HEFJNZH4VSJY8S6A",
	)
	encryptedBytes, _ := base64.StdEncoding.DecodeString(encryptedBase64)
	recorder = web.postFileHttpResponse(
		"/firmware",
		"file",
		encryptedBytes,
		map[string]string{"checksum": "77f2b19e93301391ed20a400a8bdb97185054b83e65ccc35c63f0895cbc59713"},
	)
	assert.Equal(t, 202, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "received and will be applied now")
}

func TestWeb_firmwareHandlerInvalidInput(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)

	// Wrong content type.
	recorder := web.postHttpResponseWithHeaders("/firmware", "", map[string]string{"Content-Type": "text/plain"})
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Content-Type isn't multipart/form-data")

	// Missing file.
	recorder = web.postFileHttpResponse(
		"/firmware",
		"wrongfile",
		[]byte("unencrypted firmware content\n"),
		map[string]string{},
	)
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing or invalid firmware file")

	// Missing checksum.
	recorder = web.postFileHttpResponse(
		"/firmware",
		"file",
		[]byte("unencrypted firmware content\n"),
		map[string]string{},
	)
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing or invalid checksum")

	// Wrong decryption key.
	web.firmwareDecryptionKey, _ = age.ParseX25519Identity(
		"AGE-SECRET-KEY-1XVUDFHTME5C0FUZT805RPFP7SVNPMSRVLTLJ94WHWN9ER9EW2T6QQ569A0",
	)
	encryptedBytes, _ := base64.StdEncoding.DecodeString(encryptedBase64)
	recorder = web.postFileHttpResponse(
		"/firmware",
		"file",
		encryptedBytes,
		map[string]string{"checksum": "77f2b19e93301391ed20a400a8bdb97185054b83e65ccc35c63f0895cbc59713"},
	)
	assert.Equal(t, 422, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "incorrect key or file not encrypted")

	// File not encrypted.
	web.firmwareDecryptionKey, _ = age.ParseX25519Identity(
		"AGE-SECRET-KEY-1QS7DUT0EK9LYHRXYJLLFDM26MALP78UTT48TNPZS55HEFJNZH4VSJY8S6A",
	)
	recorder = web.postFileHttpResponse(
		"/firmware",
		"file",
		[]byte("unencrypted firmware content\n"),
		map[string]string{"checksum": "77f2b19e93301391ed20a400a8bdb97185054b83e65ccc35c63f0895cbc59713"},
	)
	assert.Equal(t, 422, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "incorrect key or file not encrypted")

	// Wrong checksum.
	encryptedBytes, _ = base64.StdEncoding.DecodeString(encryptedBase64)
	recorder = web.postFileHttpResponse(
		"/firmware",
		"file",
		encryptedBytes,
		map[string]string{"checksum": "a3dfab891e82d64aeb510b1d4281ceb3c5057c7a9129957c56223a5f93d54315"},
	)
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "checksum mismatch")
}

func TestWeb_firmwareHandlerAuthorization(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)
	web.password = "mypassword"

	// Without password.
	recorder := web.postHttpResponse("/firmware", "")
	assert.Equal(t, 401, recorder.Code)

	// With wrong password.
	recorder = web.postHttpResponseWithHeaders(
		"/firmware", "", map[string]string{"Authorization": "Bearer wrongpassword"},
	)
	assert.Equal(t, 401, recorder.Code)

	// With correct password.
	recorder = web.postHttpResponseWithHeaders(
		"/firmware", "", map[string]string{"Authorization": "Bearer mypassword"},
	)
	assert.Equal(t, 400, recorder.Code)
}
