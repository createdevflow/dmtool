// Package utils provides JWT RS256 utilities for signing and verifying
// access tokens.  The private key is used to sign; the public key is used
// to verify.  Access tokens have a 15-minute TTL per Section 3.1.
package utils

import (
	"crypto/rsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims contains the claims embedded in every access token.
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed RS256 JWT with a 15-minute TTL.
func GenerateAccessToken(privateKey *rsa.PrivateKey, userID uint, email, role string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "dmtool",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// ParseAccessToken validates the token signature using the RSA public key and
// returns the embedded claims.  It rejects any token not signed with RS256.
func ParseAccessToken(publicKey *rsa.PublicKey, tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method: token must use RS256")
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// DevPrivateKey is a stable RS256 private key for local development.
const DevPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAqxmP+JXFJBopOHXcPVehCXzeqqmcns41TGfjtdy42MbSI80X
VH6nvdBM3u4g7KWFfhNzvzt0Z/nCMO65hwqwcw5Z2EBlG/NG/M+IdYGcHNS3GwIp
7T9lt2vT4GLecnAyLTLbRMTGnjGcWg+7MyCO20il0cUjbqOuQaD5J3J+6bZ8ueZc
dfCVXdmNIH+mJSBEp0Y2UQHmzaLehkTnaRVBSMtikEgw34dvVSW28yUVbpfdScRH
Lae2y51w+WGK1131d7kEYgh/8zVfSEWVMOY4k8zaJIDaG5OFfsvTsPZSnYkbpYHY
HWavc3P6IWyxBn4M9nxsj9z2PGPygJLa3W742QIDAQABAoIBAACPqGbV0SNRrKV9
tCJh0JUeVADZ2FZhON6HVrMhfyNC36yQuAYJTzIWHxX6F6D/EN5oJ5ekBuPgof4/
pb8gXDzKlsIRwcaxTYOT+Y+nxFXPNJHeRh18B1tR0LcjL25dbp2WHLmog8hACqBm
/c5aZkLhTLpfAvwghaMQ6UIQJAPc/SZNojioR++sPmh2q4tzQJ5o28nls31rGqtS
pt4rI8mrJMSLEXZezD2+mr55qrbWpN89j/jWJDwQXM9Nb54HLToQQnISf89uUAyP
5D5z+09xp96nsXx9sHXqSn/H0N0/8aetGFdhKx8+myc0CPLVkw4yVlmgNx7QzgmR
qoYIXeMCgYEAxgPSdQzVk4hvOfDhCC3mcbOqQNtGhLjmYwyxXh/tThc2/upJ2Ysh
AywQLEBwYjcII8vjndVFOaSnGOOuWuDAgX0eJEkxWQvuZSfw3kgGBRhy757+k4rk
3pW13fQAmX3R9Vx0YlMrnO2eEjkeGTVNvwGcVoRHjxe4e3fVmRcGeA8CgYEA3TQN
lry831l3/cg9aGz64hv/gKKa9p2n7rE8w0N041fn5tVdrlin3HYna6MQck0JOJqK
MV9lP9XbsmeWK1/eDy9sJpEQuYvi9NmJankuj1gO+OgWt+NhXEgy6SW5Vri5Loum
SPV+CAopmLvzGg6hYgNcfZMEAWhhqlTahloXWJcCgYEAkyVx9pNz/u9rIBZ1Uzuw
vJuzkACilaxFYCOlhnmZQeeUcbgu6t8tyAqtdgTovsOAZAF2gupiR+SoW6GOBnBv
/Ym5pWNSrYZlQHx+zN+ewZiaiKCRM5kKXLHJ9UcGTwmlxXxP7GU7yqD1Tztst3TZ
8m/C+rPP7XDGIdI6Q0pN/1cCgYEAv8xXkAD8n4jbEvCh1j7QVmfCZc4YIYfXyY5i
SkjePBXHnQ4aWYbDndPRIHJWW1VSjoLpGiBGEVCkWw+YVOmeCHqtAs+7ubaCRp2s
meaTKXVIHihXMoD0tWhgxGmSR5CJ+MLbnt1Ft0Dgim9FPa273tldRADKc1IHhQJo
LCBHp+cCgYEAmWn5ci9edgihGlkpfQzJ1XApd/OhTxw1Exnpg6ueArt295uXWSL3
ICsvoOSKz/V4fB+eBSgo0o6iD2sY8JlJ695CG2+gSxOY/SdVUzDOks4JzXlwC2Vg
hBqfD+rrOmPG1C2wVzohlOwTwgnKKgGOCEOLDZcGTSzdADah8OL2VW0=
-----END RSA PRIVATE KEY-----`

// DevPublicKey is the matching public key for DevPrivateKey.
const DevPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqxmP+JXFJBopOHXcPVeh
CXzeqqmcns41TGfjtdy42MbSI80XVH6nvdBM3u4g7KWFfhNzvzt0Z/nCMO65hwqw
cw5Z2EBlG/NG/M+IdYGcHNS3GwIp7T9lt2vT4GLecnAyLTLbRMTGnjGcWg+7MyCO
20il0cUjbqOuQaD5J3J+6bZ8ueZcdfCVXdmNIH+mJSBEp0Y2UQHmzaLehkTnaRVB
SMtikEgw34dvVSW28yUVbpfdScRHLae2y51w+WGK1131d7kEYgh/8zVfSEWVMOY4
k8zaJIDaG5OFfsvTsPZSnYkbpYHYHWavc3P6IWyxBn4M9nxsj9z2PGPygJLa3W74
2QIDAQAB
-----END PUBLIC KEY-----`


