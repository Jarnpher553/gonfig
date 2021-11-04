package types

import uuid "github.com/satori/go.uuid"

//SlaveMetaReq slave register request body
type SlaveMetaReq struct {
	ID   uuid.UUID
	Addr string
	Role string
}

//PushConfigReq push config request body
type PushConfigReq struct {
	Tag  []string
	Name string
	Body string
}

//PullConfigReq pull config request body
type PullConfigReq struct {
	Name string
	Tag  []string
}

//PullConfigResp pull config response body
type PullConfigResp struct {
	Body string
}

//SyncConfigReq sync config request body
type SyncConfigReq struct {
	Datum []*PushConfigReq
}
