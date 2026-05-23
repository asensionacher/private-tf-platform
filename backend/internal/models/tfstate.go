package models

import "time"

// TFStateLockInfo represents the lock metadata sent by Terraform on LOCK requests.
type TFStateLockInfo struct {
	ID        string    `json:"ID"`
	Operation string    `json:"Operation"`
	Info      string    `json:"Info"`
	Who       string    `json:"Who"`
	Version   string    `json:"Version"`
	Created   time.Time `json:"Created"`
	Path      string    `json:"Path"`
}

// TFStateSummary is a lightweight workspace descriptor returned by ListTFStates (UI).
// It intentionally omits the raw state blob.
type TFStateSummary struct {
	ID           string     `json:"id"`
	DeploymentID string     `json:"deployment_id"`
	Workspace    string     `json:"workspace"`
	StateSerial  int        `json:"state_serial"`
	LockID       *string    `json:"lock_id,omitempty"`
	LockedAt     *time.Time `json:"locked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
