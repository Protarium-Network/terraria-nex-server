package globals

// GetUserFriendPIDs is used by friends-scoped matchmaking searches. No
// friends system is wired up here, so this always returns an empty list -
// replace with a real lookup if you have one.
func GetUserFriendPIDs(pid uint32) []uint32 {
	return []uint32{}
}
