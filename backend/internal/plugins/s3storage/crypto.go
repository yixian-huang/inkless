package s3storage

import (
	"crypto/hmac"
	"crypto/sha256"
)

// sha256sum returns the SHA-256 hash of data.
func sha256sum(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// hmacSHA256 returns the HMAC-SHA256 of data using key.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// deriveSigningKey derives the AWS Signature Version 4 signing key.
func deriveSigningKey(secretKey, dateShort, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateShort))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
