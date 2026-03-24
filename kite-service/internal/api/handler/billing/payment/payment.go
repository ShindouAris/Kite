package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

type TransferCodeParts struct {
	AppID  string
	PlanID string
	Nonce  string
}

var transferCodeRegex = regexp.MustCompile(`(?i)\b([a-z]+)-([a-z0-9]+)-([a-z0-9_-]+)-([a-z0-9_-]+)\b`)

func VerifyHMAC(payload []byte, signature string, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	rawSignature := strings.TrimSpace(strings.ToLower(signature))
	rawSignature = strings.TrimPrefix(rawSignature, "sha256=")

	decoded, err := hex.DecodeString(rawSignature)
	if err != nil {
		return false
	}

	return hmac.Equal(expected, decoded)
}

func ParseTransferCode(description string, prefix string) (*TransferCodeParts, bool) {
	if description == "" || prefix == "" {
		return nil, false
	}

	matches := transferCodeRegex.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return nil, false
	}

	normalizedPrefix := strings.ToLower(strings.TrimSpace(prefix))
	for _, m := range matches {
		if len(m) < 5 {
			continue
		}

		if strings.ToLower(m[1]) != normalizedPrefix {
			continue
		}

		return &TransferCodeParts{
			AppID:  m[2],
			PlanID: m[3],
			Nonce:  m[4],
		}, true
	}

	return nil, false
}

func ParseAmountVND(raw string) (int, error) {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return 0, strconv.ErrSyntax
	}

	v, err := strconv.Atoi(b.String())
	if err != nil {
		return 0, err
	}

	return v, nil
}
