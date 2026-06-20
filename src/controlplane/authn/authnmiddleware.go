package authn

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"waitingroom/controlplane/authn/keyrepository"
	"waitingroom/shared/models"
	"waitingroom/shared/pg"
)

const SignatureInputHeader = "Signature-Input"
const SignatureHeader = "Signature"
const ContentDigestHeader = "Content-Digest"
const ContentTypeHeader = "Content-Type"
const ContentLengthHeader = "Content-Length"
const Algorithm = "rsa-v1_5-sha256"
const DerivedComponents = "\"@method\" \"@target-uri\" \"@authority\""
const Headers = "\"content-type\" \"content-length\" \"content-digest\""
const MaxExpiryDuration = 15 * time.Minute

var ValidDerivedComponents = []string{"@method", "@target-uri", "@authority"}
var ValidHeaderComponents = []string{"content-type", "content-length", "content-digest"}

// Global declaration prevents repeated compilation overhead

// Regex Breakdown: ^sha-512=:[A-Za-z0-9+/]{86}==:$
// -----------------------------------------------------------------------------
// ^             -> Anchor: Ensures the match begins at the absolute start of the string.
// sha-512=:     -> Literal Match: Matches the exact prefix text "sha-512=:".
// [A-Za-z0-9+/] -> Character Class: Matches any valid Base64 character (A-Z, a-z, 0-9, +, /).
// {86}          -> Quantifier: Mandates exactly 86 of the preceding Base64 characters.
// ==            -> Literal Match: Matches the two required padding equals signs.
// :             -> Literal Match: Matches the trailing colon character.
// $             -> Anchor: Ensures the match ends at the absolute end of the string.
var contentDigestRegex = regexp.MustCompile("^sha-512=:([A-Za-z0-9+/]{86}==):$")

var signatureRegex = regexp.MustCompile("^sig1=:([A-Za-z0-9+/]{342}==):$")

type SignatureInputComponent struct {
	Sig1         []string
	RawSig1      string
	Created      int64
	KeyId        string
	Algorithm    string
	RawKeyId     string
	RawAlgorithm string
	Nonce        string
	RawNonce     string
	Expires      int64
}

type Component struct {
	Method                  string
	Authority               string
	TargetUri               string
	ContentType             string
	ContentLength           string
	ContentDigest           string
	SignatureInputString    string
	Signature               string
	SignatureInputComponent SignatureInputComponent
	RawSignature            []byte
}

type ApiAuthnConfig struct {
	KeyLookUpRepository keyrepository.KeyLookup
	NonceRepository     pg.NonceRepository
}

func ApiAuthn(config ApiAuthnConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			component, err := constructComponent(r)
			if err != nil {
				message := fmt.Sprintf("Component Construction Error: %v\n", err)
				log.Printf(message)
				authFailed(w)
				return
			}

			// Verify Signature Expiration
			isSignatureExpired := verifyExpiry(r, component)
			if isSignatureExpired {
				message := fmt.Sprintf("Signature expired. Created: %v , Expired: %v\n", component.SignatureInputComponent.Created, component.SignatureInputComponent.Expires)
				log.Printf(message)
				authFailed(w)
				return
			}

			// Verify Integrity - Valid for POST/PUT/PATCH only
			integrityOk := verifyIntegrity(r, component)
			if !integrityOk {
				log.Printf("Verify Integrity failed")
				authFailed(w)
				return
			}

			// Verify signature
			signatureOk := verifySignature(r, component, &config)
			if !signatureOk {
				log.Printf("Verify Signature failed")
				authFailed(w)
				return
			}

			// Verify replay attack
			status := verifyReplayAttack(r, component, &config)
			if !status {
				log.Printf("Replay Attack detected")
				authFailed(w)
				return
			}

			// Construct User Principal
			userPrincipal := constructUserPrincipal(component)
			next.ServeHTTP(w, r.WithContext(SetUserPrincipal(r.Context(), userPrincipal)))
		})
	}
}

func constructComponent(r *http.Request) (Component, error) {
	component := Component{}
	component.Method = r.Method

	// a. Check Presence of key headers

	// Extract Signature-Input. If not present, send 401
	signatureInputHeaderValue, err := getMandatoryHeader(r, SignatureInputHeader)
	if err != nil {
		return component, err
	}
	component.SignatureInputString = signatureInputHeaderValue
	log.Printf("Read %v header ", SignatureInputHeader)

	// Extract Signature. If not present, send 401
	signatureHeaderValue, err := getMandatoryHeader(r, SignatureHeader)
	if err != nil {
		return component, err
	}
	component.Signature = signatureHeaderValue
	log.Printf("Read %v header ", SignatureHeader)

	status := ensurePrefixMatches(signatureInputHeaderValue, signatureHeaderValue)
	if !status {
		message := "Both Signature-Input and Signature must have sig1"
		return component, errors.New(message)
	}

	contentDigest := ""
	if isWritableRequest(component) {
		// Extract Content-Digest: If not present, send 401 (valid only for POST/PUT/PATCH)
		contentDigest, err = getMandatoryHeader(r, ContentDigestHeader)
		if err != nil {
			return component, err
		}
		component.ContentDigest = contentDigest
		log.Printf("Read %v header with value %v", ContentDigestHeader, contentDigest)
	}

	contentType := ""
	if isWritableRequest(component) {
		// Extract Content-Type: If not present, send 401 (valid only for POST/PUT/PATCH)
		contentType, err = getMandatoryHeader(r, ContentTypeHeader)
		if err != nil {
			return component, err
		}
		component.ContentType = contentType
		log.Printf("Read %v header with value %v", ContentTypeHeader, contentType)
	}

	contentLength := ""
	if isWritableRequest(component) {
		// Extract Content-Length: If not present, send 401 (valid only for POST/PUT/PATCH)
		contentLength, err = getMandatoryHeader(r, ContentLengthHeader)
		if err != nil {
			return component, err
		}
		component.ContentLength = contentLength
		log.Printf("Read %v header with value %v", ContentLengthHeader, contentLength)
	}

	signatureInputComponent, status := getSignatureInputComponent(component.SignatureInputString)
	if !status {
		message := "Failed to parse Signature Input"
		log.Printf(message)
		return component, errors.New(message)
	}

	// Set Derived Components: method, authority and target-uri
	component.Authority = r.Host
	component.TargetUri = getTargetUri(r)
	component.SignatureInputComponent = signatureInputComponent
	bytes, err := getRawSignature(component)
	if err != nil {
		return component, err
	}
	component.RawSignature = bytes

	return component, nil
}
func getMandatoryHeader(r *http.Request, headerName string) (string, error) {
	header := r.Header.Get(headerName)
	if header == "" {
		errorMessage := fmt.Sprintf("Header: %v not present in the request", headerName)
		log.Printf(errorMessage)
		return "", errors.New(errorMessage)
	}
	return header, nil
}

func isWritableRequest(component Component) bool {
	method := component.Method
	if method == "POST" || method == "PUT" || method == "PATCH" {
		return true
	}
	return false
}

func ensurePrefixMatches(signatureInputHeaderValue string, signatureHeaderValue string) bool {
	// Ensure Signature-Input begins with sig1=
	sig1SignatureInputPrefix, found1 := strings.CutPrefix(signatureInputHeaderValue, "sig1=")
	if !found1 {
		log.Printf("Invalid Prefix in Signature-Input: %v", sig1SignatureInputPrefix)
		return false
	}

	// Ensure Signature begins with sig1=:
	sig1SignaturePrefix, found2 := strings.CutPrefix(signatureHeaderValue, "sig1=:")

	if !found2 {
		log.Printf("Invalid Prefix in Signature-Input: %v", sig1SignaturePrefix)
		return false
	}

	return true
}

func getTargetUri(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + r.URL.RequestURI()
}

// The Verifier rejects signature which is past the expiration time in the "expires" timestamp
// Verifier rejects signature if "expires" timestamp is less than "created" timestamp
// Verifier must enforce a maximum signature lifetime. Expires must be within 15 mins and not greater. If expires > 15 mins than created, reject
func verifyExpiry(r *http.Request, component Component) bool {
	now := time.Now().Unix()
	createdTime := time.Unix(component.SignatureInputComponent.Created, 0)
	expiresTime := time.Unix(component.SignatureInputComponent.Expires, 0)

	return now > component.SignatureInputComponent.Expires ||
		component.SignatureInputComponent.Expires <= component.SignatureInputComponent.Created ||
		expiresTime.Sub(createdTime) > MaxExpiryDuration

}

func verifyIntegrity(r *http.Request, component Component) bool {
	if !isWritableRequest(component) {
		return true
	}

	contentDigestFromHeader := getContentDigestFromHeader(component.ContentDigest)
	if contentDigestFromHeader == "" {
		return false
	}
	contentDigestFromBody := getContentDigestFromBody(r)
	if contentDigestFromBody == "" {
		return false
	}

	if contentDigestFromHeader != contentDigestFromBody {
		log.Printf("Hashes failed to match: %v %v", contentDigestFromHeader, contentDigestFromBody)
		return false
	}
	return true

}

func verifySignature(r *http.Request, component Component, config *ApiAuthnConfig) bool {
	signatureInput := component.SignatureInputComponent
	if signatureInput.KeyId == "" {
		log.Printf("KeyId not present in Signature-Input")
		return false
	}

	if !isValidAlgorithm(signatureInput.Algorithm) {
		log.Printf("Invalid Algorithm present in Signature-Input: %v. Supported Algorithm: %v", signatureInput.Algorithm, Algorithm)
		return false
	}

	// Lookup public key from store using the key id value read. If key lookup fails, send 401
	if config.KeyLookUpRepository == nil {
		log.Printf("Failed to lookup key: %v as Key Lookup Repository is not configured", signatureInput.KeyId)
		return false
	}
	key, err := config.KeyLookUpRepository.GetKey(strings.ReplaceAll(signatureInput.KeyId, "\"", ""))
	if err != nil {
		log.Printf("Failed to lookup key: %v %v", signatureInput.KeyId, err)
		return false
	}
	log.Printf("Key Look up successful for: %v", signatureInput.KeyId)

	if !hasValidDerivedComponents(signatureInput) {
		log.Printf("Signature-Input doesnt have required derived components")
		return false
	}

	if isWritableRequest(component) {
		if !hasValidHeaderComponents(signatureInput) {
			log.Printf("Signature-Input doesnt have required header components")
			return false
		}
	}

	// Construct Signature Base
	signatureBase := getSignatureBase(component)

	// verify signature now
	err = verifyMessageSignature(key, []byte(signatureBase), component.RawSignature)
	if err != nil {
		return false
	}

	return true
}

func verifyMessageSignature(key *rsa.PublicKey, signatureBase []byte, signature []byte) error {
	hashed := sha256.Sum256(signatureBase)
	// VerifyPKCS1v15 parses the padding and asserts integrity against the hash
	err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return err
	}
	return nil
}

func verifyReplayAttack(r *http.Request, component Component, config *ApiAuthnConfig) bool {
	createNonceRequest := models.Nonce{
		KeyId:      component.SignatureInputComponent.KeyId,
		NonceValue: component.SignatureInputComponent.Nonce,
		CreatedAt:  time.Now().UTC(),
	}

	if config.NonceRepository == nil {
		log.Printf("Failed to verify replay attack: %v as Nonce Repository is not configured", component.SignatureInputComponent.KeyId)
		return false
	}
	isOk, err := config.NonceRepository.TryUseNonce(r.Context(), createNonceRequest)
	if !isOk || err != nil {
		log.Printf("Nonce Persistence failed: %v", err)
		return false
	}
	return true
}

func getSignatureBase(component Component) string {
	var output = ""
	methodStr := fmt.Sprintf("\"@method\": %v\n", component.Method)
	targetUriStr := fmt.Sprintf("\"@target-uri\": %v\n", component.TargetUri)
	authorityStr := fmt.Sprintf("\"@authority\": %v\n", component.Authority)
	signatureParamsStr := fmt.Sprintf("\"@signature-params\": %v", component.SignatureInputComponent.RawSig1)
	if !isWritableRequest(component) {
		output = fmt.Sprintf("%v%v%v%v", methodStr, targetUriStr, authorityStr, signatureParamsStr)
	} else {
		contentTypeStr := fmt.Sprintf("\"content-type\": %v\n", component.ContentType)
		contentLengthStr := fmt.Sprintf("\"content-length\": %v\n", component.ContentLength)
		contentDigestStr := fmt.Sprintf("\"content-digest\": %v\n", component.ContentDigest)

		output = fmt.Sprintf("%v%v%v%v%v%v%v", methodStr, targetUriStr, authorityStr, contentTypeStr, contentLengthStr, contentDigestStr, signatureParamsStr)
	}
	return output
}

func getSignatureInputComponent(signatureInputHeaderValue string) (SignatureInputComponent, bool) {
	// Extract keyid from Signature-Input. If not present, send 401
	splits := strings.Split(signatureInputHeaderValue, ";")
	signatureInput := SignatureInputComponent{}
	createdFound := false
	expiresFound := false
	signatureInput.RawSig1 = strings.TrimPrefix(signatureInputHeaderValue, "sig1=")
	for _, value := range splits {
		innerSplits := strings.Split(value, "=")
		if len(innerSplits) == 0 {
			continue
		}
		if len(innerSplits) != 2 {
			// invalid signature input, bail out
			log.Printf("Invalid Signature Input value: %v, %v", signatureInputHeaderValue, innerSplits)
			return SignatureInputComponent{}, false
		}
		key := innerSplits[0]
		val := innerSplits[1]
		if key == "keyid" {
			signatureInput.KeyId = strings.ReplaceAll(val, "\"", "")
			signatureInput.RawKeyId = val
		} else if key == "alg" {
			signatureInput.Algorithm = strings.ReplaceAll(val, "\"", "")
			signatureInput.RawAlgorithm = val
		} else if key == "created" {
			num, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				log.Printf("Error during conversion of created: %v", err)
				return SignatureInputComponent{}, false
			}
			signatureInput.Created = num
			createdFound = true
		} else if key == "expires" {
			num, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				log.Printf("Error during conversion of expires: %v", err)
				return SignatureInputComponent{}, false
			}
			signatureInput.Expires = num
			expiresFound = true
		} else if key == "nonce" {
			signatureInput.Nonce = strings.ReplaceAll(val, "\"", "")
			signatureInput.RawNonce = val
		} else if key == "sig1" {
			s := strings.ReplaceAll(val, "(", "")
			s2 := strings.ReplaceAll(s, ")", "")
			signatureInput.Sig1 = strings.Split(s2, " ")
		}
	}

	if signatureInput.KeyId == "" || signatureInput.Algorithm == "" || createdFound == false || signatureInput.Nonce == "" || len(signatureInput.Sig1) == 0 || expiresFound == false {
		return signatureInput, false
	}
	return signatureInput, true
}

func isValidAlgorithm(algorithm string) bool {
	if algorithm == "" || algorithm != Algorithm {
		return false
	}
	return true
}

func hasValidDerivedComponents(signatureInput SignatureInputComponent) bool {
	// 3 components must be present: "@method" "@target-uri" "@authority". If not, return 401

	// Check if all elements of ValidDerivedComponents are present in signatureInput.Sig1
	for _, value := range ValidDerivedComponents {
		foundVal := false
		for _, val := range signatureInput.Sig1 {
			if value == strings.ReplaceAll(val, "\"", "") {
				foundVal = true
			}
		}
		if !foundVal {
			return false
		}
	}
	return true
}

func hasValidHeaderComponents(signatureInput SignatureInputComponent) bool {
	// 3 components must be present: ""content-type", "content-length", "content-digest"". If not, return 401

	// Check if all elements of ValidHeaderComponents are present in signatureInput.Sig1
	for _, value := range ValidHeaderComponents {
		foundVal := false
		for _, val := range signatureInput.Sig1 {
			if value == strings.ReplaceAll(val, "\"", "") {
				foundVal = true
			}
		}
		if !foundVal {
			return false
		}
	}
	return true
}

// Extract SHA512 hash from Content-Digest request header. It is of this form: sha-512=:<hash>:
func getContentDigestFromHeader(contentDigestHeaderValue string) string {
	// FindStringSubmatch returns: [full_match, captured_hash]
	matches := contentDigestRegex.FindStringSubmatch(contentDigestHeaderValue)
	if len(matches) < 2 {
		return ""
	}
	extractedContentDigestValue := matches[1]
	return extractedContentDigestValue
}

func getRawSignature(component Component) ([]byte, error) {
	// FindStringSubmatch returns: [full_match, captured_hash]
	matches := signatureRegex.FindStringSubmatch(component.Signature)
	if len(matches) < 2 {
		return nil, errors.New("Failed to extract signature")
	}
	base64EncodedSignature := matches[1]
	bytes, err := base64.StdEncoding.DecodeString(base64EncodedSignature)
	if err != nil {
		return nil, errors.New("Failed to decode signature")
	}
	return bytes, nil
}

func getContentDigestFromBody(r *http.Request) string {
	// Extract request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		return ""
	}
	bodyHash := sha512.Sum512([]byte(bodyBytes))
	base64BodyHash := base64.StdEncoding.EncodeToString(bodyHash[:])

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return base64BodyHash
}

func authFailed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func constructUserPrincipal(component Component) UserPrincipal {
	userPrincipal := UserPrincipal{}
	userPrincipal.Id = component.SignatureInputComponent.KeyId
	userPrincipal.AuthenticationMethod = "HTTP_MESSAGE_SIGNATURE"
	return userPrincipal
}
