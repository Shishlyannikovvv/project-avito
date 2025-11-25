package domain

import "errors"

var (
	// Ошибки не найденных сущностей
	ErrUserNotFound = errors.New("user not found")
	ErrTeamNotFound = errors.New("team not found")
	ErrPRNotFound   = errors.New("pull request not found")

	// Ошибки бизнес-логики
	ErrPRAlreadyMerged   = errors.New("pull request already merged")
	ErrReviewerNotActive = errors.New("reviewer is not active")
	ErrNoReviewersFound  = errors.New("no eligible reviewers found")
)
