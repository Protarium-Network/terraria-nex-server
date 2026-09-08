// terraria-nex-server is a NEX server for Terraria (Wii U), built from
// scratch: no reference server implementation for this title exists
// anywhere (unlike most other Wii U titles Pretendo has full or partial
// servers for). Terraria's own binary (RPX) analysis shows it links
// Nintendo's official nn::nex SDK wrapper using only the standard, generic
// MatchMaking/MatchmakeExtension/NATTraversal/Utility protocol set - no
// bespoke per-title RMC protocol - so this server sticks to that same
// generic set. See globals/password_from_pid.go for the account model:
// PIDs are authenticated with a shared HMAC secret instead of a local
// account database or an account gRPC service, so this server can accept
// any PID that was already verified elsewhere (e.g. by whatever issues the
// NEX token for it).
package main

import (
	"database/sql"
	"encoding/hex"
	"os"
	"strconv"
	"strings"

	"terraria-nex-server/globals"
	nexpkg "terraria-nex-server/nex"

	_ "github.com/lib/pq"

	"github.com/PretendoNetwork/plogger-go"
	"github.com/joho/godotenv"
)

func main() {
	globals.Logger = plogger.NewLogger()

	if err := godotenv.Load(); err != nil {
		globals.Logger.Warning("Error loading .env file")
	}

	kerberosPassword := strings.TrimSpace(os.Getenv("PN_TERRARIA_KERBEROS_PASSWORD"))
	if kerberosPassword == "" {
		globals.Logger.Warningf("PN_TERRARIA_KERBEROS_PASSWORD environment variable not set. Using default password: %q", globals.KerberosPassword)
	} else {
		globals.KerberosPassword = kerberosPassword
	}
	globals.InitAccounts()

	secretHex := strings.TrimSpace(os.Getenv("PN_TERRARIA_NEX_PASSWORD_SECRET"))
	if len(secretHex) < 64 {
		globals.Logger.Critical("PN_TERRARIA_NEX_PASSWORD_SECRET must be at least 32 bytes encoded as hexadecimal")
		os.Exit(1)
	}
	secret, err := hex.DecodeString(secretHex)
	if err != nil {
		globals.Logger.Criticalf("PN_TERRARIA_NEX_PASSWORD_SECRET is not valid hex: %v", err)
		os.Exit(1)
	}
	globals.NEXPasswordSecret = secret

	authPort := strings.TrimSpace(os.Getenv("PN_TERRARIA_AUTHENTICATION_SERVER_PORT"))
	if authPort == "" {
		globals.Logger.Critical("PN_TERRARIA_AUTHENTICATION_SERVER_PORT environment variable not set")
		os.Exit(1)
	}
	if _, err := strconv.Atoi(authPort); err != nil {
		globals.Logger.Criticalf("PN_TERRARIA_AUTHENTICATION_SERVER_PORT is not a valid port: %s", authPort)
		os.Exit(1)
	}

	secureHost := strings.TrimSpace(os.Getenv("PN_TERRARIA_SECURE_SERVER_HOST"))
	if secureHost == "" {
		globals.Logger.Critical("PN_TERRARIA_SECURE_SERVER_HOST environment variable not set")
		os.Exit(1)
	}

	securePort := strings.TrimSpace(os.Getenv("PN_TERRARIA_SECURE_SERVER_PORT"))
	if securePort == "" {
		globals.Logger.Critical("PN_TERRARIA_SECURE_SERVER_PORT environment variable not set")
		os.Exit(1)
	}
	if _, err := strconv.Atoi(securePort); err != nil {
		globals.Logger.Criticalf("PN_TERRARIA_SECURE_SERVER_PORT is not a valid port: %s", securePort)
		os.Exit(1)
	}

	globals.Postgres, err = sql.Open("postgres", os.Getenv("PN_TERRARIA_POSTGRES_URI"))
	if err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}
	if err := globals.Postgres.Ping(); err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}
	globals.Logger.Success("Connected to Postgres!")

	go nexpkg.StartAuthenticationServer()
	go nexpkg.StartSecureServer()
	select {}
}
