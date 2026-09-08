package nex

import (
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"terraria-nex-server/globals"

	nexgo "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	common_matchmaking "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making"
	common_matchmaking_ext "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making-ext"
	common_matchmake_extension "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension"
	common_nat_traversal "github.com/PretendoNetwork/nex-protocols-common-go/v2/nat-traversal"
	common_secure "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	common_utility "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility"
	matchmaking "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	matchmaking_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
	matchmaking_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/v2/nat-traversal"
	secure "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
)

func StartSecureServer() {
	globals.SecureServer = nexgo.NewPRUDPServer()

	globals.SecureEndpoint = nexgo.NewPRUDPEndPoint(1)
	globals.SecureEndpoint.IsSecureEndPoint = true
	globals.SecureEndpoint.ServerAccount = globals.SecureServerAccount
	globals.SecureEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	globals.SecureEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	globals.SecureServer.BindPRUDPEndPoint(globals.SecureEndpoint)

	globals.SecureServer.LibraryVersions.SetDefault(nexgo.NewLibraryVersion(3, 8, 0))
	globals.SecureServer.AccessKey = AccessKey
	globals.SecureServer.ByteStreamSettings.UseStructureHeader = true

	globals.SecureEndpoint.OnData(func(packet nexgo.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[Secure] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
	})

	globals.SecureEndpoint.OnConnectionEnded(func(connection *nexgo.PRUDPConnection) {
		fmt.Printf("[Secure] PID=%d disconnected\n", uint64(connection.PID()))
	})

	globals.MatchmakingManager = common_globals.NewMatchmakingManager(globals.SecureEndpoint, globals.Postgres)
	globals.MatchmakingManager.GetUserFriendPIDs = globals.GetUserFriendPIDs

	registerSecureServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_TERRARIA_SECURE_SERVER_PORT"))
	globals.Logger.Successf("[Terraria] Secure server listening on UDP %d", port)
	globals.SecureServer.Listen(port)
}

func registerSecureServerProtocols() {
	secureProtocol := secure.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(secureProtocol)
	secureCommon := common_secure.NewCommonProtocol(secureProtocol)
	secureCommon.EnableInsecureRegister() // Game uses TicketGranting::LoginEx
	secureCommon.CreateReportDBRecord = func(pid types.PID, reportID types.UInt32, reportData types.QBuffer) error {
		globals.Logger.Warningf("Player report from PID %d (report ID %d, %d bytes) discarded: this server does not store reports",
			pid, reportID, len(reportData))
		return nil
	}

	utilityProtocol := utility.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(utilityProtocol)
	commonUtilityProtocol := common_utility.NewCommonProtocol(utilityProtocol)
	commonUtilityProtocol.GenerateNEXUniqueID = generateNEXUniqueID

	// NAT Traversal + matchmaking - real online world hosting/joining.
	// No DataStore or Ranking registration: nothing in the RPX's linked
	// symbols or strings pointed at either being used by this title.
	natTraversalProtocol := nat_traversal.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(natTraversalProtocol)
	common_nat_traversal.NewCommonProtocol(natTraversalProtocol)

	matchMakingProtocol := matchmaking.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchMakingProtocol)
	commonMatchMakingProtocol := common_matchmaking.NewCommonProtocol(matchMakingProtocol)
	commonMatchMakingProtocol.SetManager(globals.MatchmakingManager)
	// Not wired up by SetManager - see get_detailed_participants.go.
	matchMakingProtocol.SetHandlerGetDetailedParticipants(getDetailedParticipants)

	matchMakingExtProtocol := matchmaking_ext.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchMakingExtProtocol)
	commonMatchMakingExtProtocol := common_matchmaking_ext.NewCommonProtocol(matchMakingExtProtocol)
	commonMatchMakingExtProtocol.SetManager(globals.MatchmakingManager)

	matchmakeExtensionProtocol := matchmake_extension.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol := common_matchmake_extension.NewCommonProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol.SetManager(globals.MatchmakingManager)
	// Required by AutoMatchmakePostpone/AutoMatchmakeWithSearchCriteriaPostpone
	// ("find an opponent" calls) - both hard-fail with Core::NotImplemented
	// if left nil.
	commonMatchmakeExtensionProtocol.CleanupMatchmakeSessionSearchCriterias = func(searchCriterias types.List[matchmaking_types.MatchmakeSessionSearchCriteria]) {}
	commonMatchmakeExtensionProtocol.CleanupSearchMatchmakeSession = func(matchmakeSession *matchmaking_types.MatchmakeSession) {}
}

// nexUniqueIDCounter is seeded from the current time so IDs stay unique
// across server restarts too, not just within a single run.
var nexUniqueIDCounter = func() *atomic.Uint64 {
	var counter atomic.Uint64
	counter.Store(uint64(time.Now().UnixNano()))
	return &counter
}()

func generateNEXUniqueID() uint64 {
	return nexUniqueIDCounter.Add(1)
}
