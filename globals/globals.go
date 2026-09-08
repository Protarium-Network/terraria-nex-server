package globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/plogger-go"
)

var Postgres *sql.DB
var MatchmakingManager *common_globals.MatchmakingManager

var Logger *plogger.Logger
var KerberosPassword = "password" // * Default password

// NEXPasswordSecret is shared with whatever issues this title's NEX token
// (see PasswordFromPID) - both sides derive the same per-PID password
// independently, without a local account database or an account gRPC
// service to call.
var NEXPasswordSecret []byte

var AuthenticationServer *nex.PRUDPServer
var AuthenticationEndpoint *nex.PRUDPEndPoint

var SecureServer *nex.PRUDPServer
var SecureEndpoint *nex.PRUDPEndPoint
