package vars

import uuid "github.com/satori/go.uuid"

type Role string

const (
	RoleMaster = "Master"
	RoleSlave  = "Slave"
)

type ServerMetadata struct {
	ID    uuid.UUID
	Role  Role
	LAddr string
	RAddr string
}
