package vars

import uuid "github.com/satori/go.uuid"

type SlaveMetaReq struct {
	ID   uuid.UUID
	Addr string
	Role string
}

type PushConfigReq struct {
	Tag  []string
	Name string
	Body string
}

type PullConfigReq struct {
	Name string
	Tag  []string
}

type PullConfigResp struct {
	Body string
}

type SyncConfigReq struct {
	Datum []*PushConfigReq
}
