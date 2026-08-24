// Package ctxkeys defines string constants for gin.Context key names.
package ctxkeys

const (
	UserID       = "userID"
	UserEmail    = "userEmail"
	IsSuperAdmin = "isSuperAdmin"
	OrgID        = "orgID"
	// SessionIDHash is the sha256 hex of the caller's raw session token.
	// Populated by the auth middleware so handlers like /me/password can
	// scope "keep this session, delete the others" correctly.
	SessionIDHash = "sessionIDHash"
	// ProblemCode and ProblemDetail carry the error code and the English
	// detail sentence of the problem document a request was refused with.
	// problem.WriteProblem sets both on its way out so middleware that runs
	// *around* the refusal can still tell what the refusal was.
	//
	// The audit-denial middleware is the reason they exist: it sees only a
	// 403 on the response writer, and "forbidden", "superadmin access
	// required", "organization context required" and a CSRF rejection are
	// four very different events to an investigator.
	ProblemCode   = "problemCode"
	ProblemDetail = "problemDetail"
)
