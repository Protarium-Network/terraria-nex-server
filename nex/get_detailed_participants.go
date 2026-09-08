package nex

import (
	"strconv"

	"terraria-nex-server/globals"

	"github.com/lib/pq"

	nexgo "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	matchmaking "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	matchmaking_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// getDetailedParticipants implements MatchMaking::GetDetailedParticipants,
// left unimplemented by nex-protocols-common-go. A client that never gets a
// real response here (Core::NotImplemented instead) can hang indefinitely
// trying to join a session, since it can't populate the lobby's participant
// list.
//
// Participant PIDs come straight from matchmaking.gatherings.participants
// (a numeric(10,0)[] column the shared common_globals.MatchmakingManager
// already keeps up to date on join/leave).
func getDetailedParticipants(err error, packet nexgo.PacketInterface, callID uint32, idGathering types.UInt32) (*nexgo.RMCMessage, *nexgo.Error) {
	if err != nil {
		return nil, nexgo.NewError(nexgo.ResultCodes.Core.InvalidArgument, err.Error())
	}

	var participantPIDs pq.Int64Array
	row := globals.Postgres.QueryRow(
		"SELECT participants::bigint[] FROM matchmaking.gatherings WHERE id = $1",
		uint32(idGathering),
	)
	if scanErr := row.Scan(&participantPIDs); scanErr != nil {
		return nil, nexgo.NewError(nexgo.ResultCodes.Core.Exception, "change_error")
	}

	participants := types.NewList[matchmaking_types.ParticipantDetails]()
	for _, pid := range participantPIDs {
		details := matchmaking_types.NewParticipantDetails()
		details.IDParticipant = types.NewPID(uint64(pid))
		details.StrName = types.NewString(strconv.FormatInt(pid, 10))
		details.StrMessage = types.NewString("")
		details.UIParticipants = types.NewUInt16(1)
		participants = append(participants, details)
	}

	endpoint := packet.Sender().Endpoint()
	rmcResponseStream := nexgo.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	participants.WriteTo(rmcResponseStream)

	rmcResponse := nexgo.NewRMCSuccess(endpoint, rmcResponseStream.Bytes())
	rmcResponse.ProtocolID = matchmaking.ProtocolID
	rmcResponse.MethodID = matchmaking.MethodGetDetailedParticipants
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
