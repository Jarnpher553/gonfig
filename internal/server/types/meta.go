package types

import uuid "github.com/satori/go.uuid"

type Role string

const (
	//RoleMaster roleMaster
	RoleMaster = "Master"
	//RoleSlave roleSlave
	RoleSlave = "Slave"
)

//Slaves slaves
type Slaves = []*ServerMetadata

//ServerMetadata server metadata
type ServerMetadata struct {
	ID    uuid.UUID
	Role  Role
	LAddr string
	RAddr string
}

//ConfigMetadata config metadata
type ConfigMetadata struct {
	Name string
	Tags []string
}
