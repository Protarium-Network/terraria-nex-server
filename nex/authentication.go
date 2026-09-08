// Package nex is a NEX server for Terraria (Wii U port).
// game_server_id 10198f00 / 270110464 decimal, title 0005000010198f00 -
// AccessKey and build/branch strings from kinnay.github.io's public Wii U
// NEX game database (data/nexwiiu.json). Terraria's own binary (RPX)
// analysis (decompressed .rodata string scan) shows it links Nintendo's
// official nn::nex SDK wrapper with the STANDARD, generic protocol set only
// - nn::nex::MatchMakingClient, MatchmakeExtensionClient, RankingClient,
// UtilityClient, JobNNIDLogin/JobCafeLogin - and no custom/game-specific RMC
// protocol names anywhere in the binary. This is the same "vanilla nn::nex
// session hosting" pattern used by many third-party (non-first-party) Wii U
// ports, unlike titles with bespoke protocols.
//
// No real console capture exists for this title (no reference server ever
// existed for it) - LibraryVersion (3, 8, 0) is a first guess from the
// branch string ("release/ngs/3.8.x.200x") and may need adjustment.
package nex

import (
	"fmt"
	"os"
	"strconv"

	"terraria-nex-server/globals"

	nexgo "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

// AccessKey from kinnay.github.io's public Wii U NEX game database
// (data/nexwiiu.json, "Terraria" entry, game_server_id 270110464 = 0x10198f00).
const AccessKey = "3d37fbdb"

func StartAuthenticationServer() {
	globals.AuthenticationServer = nexgo.NewPRUDPServer()

	globals.AuthenticationEndpoint = nexgo.NewPRUDPEndPoint(1)
	globals.AuthenticationEndpoint.ServerAccount = globals.AuthenticationServerAccount
	globals.AuthenticationEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	globals.AuthenticationEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	globals.AuthenticationServer.BindPRUDPEndPoint(globals.AuthenticationEndpoint)

	globals.AuthenticationServer.LibraryVersions.SetDefault(nexgo.NewLibraryVersion(3, 8, 0))
	globals.AuthenticationServer.AccessKey = AccessKey
	globals.AuthenticationServer.ByteStreamSettings.UseStructureHeader = true

	globals.AuthenticationEndpoint.OnData(func(packet nexgo.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[Auth] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
	})

	registerAuthenticationServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_TERRARIA_AUTHENTICATION_SERVER_PORT"))
	globals.Logger.Successf("[Terraria] Authentication server listening on UDP %d", port)
	globals.AuthenticationServer.Listen(port)
}

func registerAuthenticationServerProtocols() {
	ticketGrantingProtocol := ticket_granting.NewProtocol()
	globals.AuthenticationEndpoint.RegisterServiceProtocol(ticketGrantingProtocol)
	commonTicketGrantingProtocol := common_ticket_granting.NewCommonProtocol(ticketGrantingProtocol)

	securePort, _ := strconv.Atoi(os.Getenv("PN_TERRARIA_SECURE_SERVER_PORT"))
	secureHost := os.Getenv("PN_TERRARIA_SECURE_SERVER_HOST")

	secureStationURL := types.NewStationURL("")
	secureStationURL.SetURLType(constants.StationURLPRUDPS)
	secureStationURL.SetAddress(secureHost)
	secureStationURL.SetPortNumber(uint16(securePort))
	secureStationURL.SetConnectionID(1)
	secureStationURL.SetPrincipalID(types.NewPID(2))
	secureStationURL.SetStreamID(1)
	secureStationURL.SetStreamType(constants.StreamTypeRVSecure)
	secureStationURL.SetType(uint8(constants.StationURLFlagPublic))

	// No account-ban/moderation gate wired up here - accept any login whose
	// signature already checked out via AccountDetailsByUsername/ByPID
	// above. Replace with real validation if you need one.
	commonTicketGrantingProtocol.ValidateLoginData = func(pid types.PID, loginData types.DataHolder) *nexgo.Error {
		return nil
	}
	commonTicketGrantingProtocol.SecureStationURL = secureStationURL
	// Build string from kinnay.github.io's database entry ("build:3_8_13_2004_0").
	commonTicketGrantingProtocol.BuildName = types.NewString("3_8_13_2004_0")
	commonTicketGrantingProtocol.SecureServerAccount = globals.SecureServerAccount
}
