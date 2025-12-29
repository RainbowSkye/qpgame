package response

import (
	"common"
	"hall/models/request"
)

type UpdateUserAddressResp struct {
	common.Result
	UpdateUserData request.UpdateUserAddressReq `json:"updateUserData"`
}
