package api

// Exported lightweight aliases let the service layer consume decoded API values
// without making transport field names part of the public CLI model.
type UserDTO = apiUser
type RoomDTO = apiRoom
type MessageDTO = apiMessage
