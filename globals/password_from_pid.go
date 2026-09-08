package globals

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// PasswordFromPID derives the NEX password directly from the PID using a
// shared secret (NEXPasswordSecret), instead of looking the account up
// through an account gRPC service. This lets this server authenticate any
// PID that was already verified elsewhere (e.g. by whatever issues the NEX
// token) without needing its own local account database - both sides just
// need to agree on the same secret.
func PasswordFromPID(pid types.PID) (string, uint32) {
	if len(NEXPasswordSecret) < 32 {
		Logger.Error("NEXPasswordSecret must be at least 32 bytes")
		return "", nex.ResultCodes.RendezVous.InvalidUsername
	}

	pidBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(pidBytes, uint64(pid))
	mac := hmac.New(sha256.New, NEXPasswordSecret)
	_, _ = mac.Write(pidBytes)

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), 0
}
