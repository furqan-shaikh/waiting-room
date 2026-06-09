package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log"
	"time"
)

const DerivedComponents = "\"@method\" \"@target-uri\" \"@authority\""
const Headers = "\"content-type\" \"content-length\" \"content-digest\""
const Body = "{\"originApplication\": \"cloud.oracle.com\", \"maxActiveUsersCount\": 1}"
const KeyId = "key-1"
const Alg = "rsa-v1_5-sha256"
const KeySize = 2048

type Key struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func getSignatureBase(keyId string, created int64, method string, targetUri string, authority string, contentType string, body string, alg string) string {
	var output = ""
	methodStr := fmt.Sprintf("\"@method\": %v\n", method)
	targetUriStr := fmt.Sprintf("\"@target-uri\": %v\n", targetUri)
	authorityStr := fmt.Sprintf("\"@authority\": %v\n", authority)
	contentTypeStr := fmt.Sprintf("\"content-type\": %v\n", contentType)
	contentLengthStr := fmt.Sprintf("\"content-length\": %v\n", len([]byte(body)))
	// take the hash of the body
	bodyHash := sha512.Sum512([]byte(body))
	base64BodyHash := fmt.Sprintf("sha-512=:%v:", base64.StdEncoding.EncodeToString(bodyHash[:]))
	contentDigestStr := fmt.Sprintf("\"content-digest\": %v\n", base64BodyHash)
	signatureParamsStr := fmt.Sprintf("\"@signature-params\": %v", getSig1Value(keyId, created, alg))

	output = fmt.Sprintf("%v%v%v%v%v%v%v", methodStr, targetUriStr, authorityStr, contentTypeStr, contentLengthStr, contentDigestStr, signatureParamsStr)

	fmt.Println(output)

	return output
}

func getSig1Value(keyId string, created int64, alg string) string {
	output := ""
	componentsStr := fmt.Sprintf("(%v %v);", DerivedComponents, Headers)

	// Create created
	createdStr := fmt.Sprintf("created=%v;", created)

	// Create keyid
	keyIdStr := fmt.Sprintf("keyid=\"%v\";", keyId)

	// Create alg
	algStr := fmt.Sprintf("alg=\"%v\";", alg)

	output = fmt.Sprintf("%v%v%v%v", componentsStr, createdStr, keyIdStr, algStr)

	return output
}

func getSignatureInput(keyId string, created int64, alg string) []byte {
	// Prefix sig1
	return []byte(fmt.Sprintf("sig1=%v", getSig1Value(keyId, created, alg)))
}

func getSignatureInRFC9421(signature []byte) string {
	// convert to base64
	base64 := base64.StdEncoding.EncodeToString(signature)
	return fmt.Sprintf("sig1=:%v:", base64)
}

func getKeyPair(keySize int) (Key, error) {
	// Generate a secure RSA Key Pair (Minimum 2048 bits recommended)
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return Key{}, err
	}
	publicKey := &privateKey.PublicKey
	return Key{PrivateKey: privateKey, PublicKey: publicKey}, nil
}

func generateSignature(key Key, signatureBase []byte) ([]byte, error) {
	// Compute the SHA-256 hash of the payload. The native Go RSA functions require the pre-computed hash digest
	hashed := sha256.Sum256(signatureBase)

	// SignPKCS1v15 applies RSASSA-PKCS1-v1_5 padding and signs the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, key.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	fmt.Printf("Signature generated successfully. Length: %d bytes\n", len(signature))
	return signature, nil
}

func verifySignature(key Key, signatureBase []byte, signature []byte) error {
	hashed := sha256.Sum256(signatureBase)
	// VerifyPKCS1v15 parses the padding and asserts integrity against the hash
	err := rsa.VerifyPKCS1v15(key.PublicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	signatureBase := getSignatureBase(KeyId, time.Now().Unix(), "POST", "http://localhost:3000/waitingRooms", "localhost:3000", "application/json", Body, Alg)
	key, err := getKeyPair(KeySize)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	signature, err := generateSignature(key, []byte(signatureBase))
	if err != nil {
		log.Fatalf("Error generating signature: %v", err)
	}
	fmt.Println(getSignatureInRFC9421(signature))

	verifyErr := verifySignature(key, []byte(signatureBase), signature)
	if verifyErr != nil {
		log.Fatalf("Signature Verification Failed: %v", verifyErr)
	}
	fmt.Println("Signature Verification successful!!")

	signatureBaseDifferent := getSignatureBase(KeyId, time.Now().Unix(), "GET", "http://localhost:3000/waitingRooms", "localhost:3000", "application/json", Body, Alg)
	verifyErr = verifySignature(key, []byte(signatureBaseDifferent), signature)
	if verifyErr != nil {
		log.Fatalf("Signature Verification Failed: %v", verifyErr)
	}
	fmt.Println("Signature Verification successful!!")
}

// 	// 2. Define the message payload
// 	message := []byte("Actionable payload requiring data authentication")

// 	// ==========================================
// 	// VERIFICATION OPERATION (Using Public Key)
// 	// ==========================================

// }
