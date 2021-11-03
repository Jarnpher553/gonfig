package types

import uuid "github.com/satori/go.uuid"

type Role string

const (
	RoleMaster = "Master"
	RoleSlave  = "Slave"
)

type Slaves = []*ServerMetadata

type ServerMetadata struct {
	ID    uuid.UUID
	Role  Role
	LAddr string
	RAddr string
}

type ConfigMeta struct {
	Name string
	Tags []string
}
