package types

import "fmt"

/* Submission represents a single submission from any of the "SourceSystem."
 * The fields contain references to image source locations and submission related metadata.
 */
type Submission struct = {
	SubmissionAuthor string
	SubmissionContactDetails string
	SourceSystem SourceSystem
	ImageLocation string
	SubmissionConnection string
	SubmissionStatus SubmissionStatus
}

/* Source System Represents the source for the Submission structs.
 *
 * Is a constant int ranging 0 - 3 which formats as strings:
 * Unknown, Web, Email, and SMS respectively.
 */

type SourceSystem int
const (
	SourceUnknown SourceSystem = iota
	SourceWeb // Recall that QR code routes to webapp
	SourceEmail
	SourceSms
)

func (s SourceSystem) String() string {
	switch s {
		case SourceUnkown: return "Unknown"
		case SourceWeb: return "Web"
		case SourceEmail: return "Email"
		case SourceSms: return "SMS"
	}
}

/* SubmissionStatus is an enum representing the status states
 *
 * Is a const int ranging from 0 - 4 which formats as strings
 * Unknown Failure, Moderation Pending, Moderation Rejected,
 * Moderation Accepted, and Complete respectively
 *
 */

type SubmissionStatus int
const (
	SubmissionUnkownFailure SubmissionStatus = iota
	SubmissionModerationPending
	SubmissionModerationRejected
	SubmissionModerationAccepted
	SubmissionComplete
)

func (s SubmissionStatus) String() string {
	switch s {
		case SubmissionUnkownFailure: return "Unknown Failure"
		case SubmissionModerationPending: return "Moderation Pending"
		case SubmissionModerationRejected: return "Moderation Rejected"
		case SubmissionModerationAccepted: return "Moderation Accepted"
		case SubmissionComplete: return "Complete"
	}
}
