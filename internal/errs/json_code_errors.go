package errs

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorCode string

// Объявляем набор констант, представляющих допустимые коды ошибок.
const (
	CodeTeamExists  ErrorCode = "TEAM_EXISTS"
	CodePRExists    ErrorCode = "PR_EXISTS"
	CodePRMerged    ErrorCode = "PR_MERGED"
	CodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	CodeNoCandidate ErrorCode = "NO_CANDIDATE"
	CodeNotFound    ErrorCode = "NOT_FOUND"
)

var (
	ExistError       error = fmt.Errorf("This entity alreay exist")
	NotFountError    error = fmt.Errorf("This entity not found")
	PRMergedError    error = fmt.Errorf("Merged error")
	NotAssignedError error = fmt.Errorf("User not assigned")
	NO_CANDIDATE     error = fmt.Errorf("No condidate")
	NoCandidateError error = fmt.Errorf(":)")
)

type ErrorResponse struct {
	Error struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
	} `json:"error"`
}

func JsonCodeResp(w http.ResponseWriter, errorCode ErrorCode, msg string, httpStatus int) {
	response := ErrorResponse{}
	response.Error.Code = errorCode
	response.Error.Message = msg

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println(response)
	}
}
