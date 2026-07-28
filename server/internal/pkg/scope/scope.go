// Package scope narrows what rows a principal can reach, on top of the tenant
// scoping every query already does via customer_id.
//
// A tutor or student is a dashboard_users row like anyone else — their id is
// what batch_tutors.tutor_id / batch_students.student_id reference, since
// tutors.dashboard_user_id / students.dashboard_user_id are those tables' own
// primary keys (see migration 000012). So restricting a query to "my batches"
// needs nothing beyond the caller's user id: no subject-type JWT claim, no
// per-request lookup of what kind of principal they are — just their id, joined
// against tutors/students at query time.
package scope

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"tutorpilot/internal/middleware"
)

type Scope struct {
	CustomerID int
	UserID     int
}

func FromContext(c *gin.Context) Scope {
	userID, _ := strconv.Atoi(c.GetString(middleware.CtxUserID))
	return Scope{CustomerID: middleware.CustomerID(c), UserID: userID}
}

// BatchPredicate returns a SQL fragment restricting results to the batches this
// principal belongs to, plus the argument it references. batchIDExpr is the
// expression naming the batch id in the surrounding query, e.g. "l.batch_id".
// nextArg is the next free placeholder number; the same value is reused for
// every reference in the fragment, so exactly one argument is returned.
//
// An admin — someone who is neither a tutor nor a student — is not restricted
// at all. Both EXISTS checks against batch_tutors/batch_students ride the
// UNIQUE(batch_id, tutor_id) / UNIQUE(batch_id, student_id) constraints that
// already exist on those tables.
func (s Scope) BatchPredicate(batchIDExpr string, nextArg int) (string, []any) {
	fragment := fmt.Sprintf(
		` AND (
			EXISTS (SELECT 1 FROM batch_tutors bt WHERE bt.batch_id = %[1]s AND bt.tutor_id = $%[2]d)
			OR EXISTS (SELECT 1 FROM batch_students bs WHERE bs.batch_id = %[1]s AND bs.student_id = $%[2]d)
			OR (
				NOT EXISTS (SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $%[2]d)
				AND NOT EXISTS (SELECT 1 FROM students st WHERE st.dashboard_user_id = $%[2]d)
			)
		)`,
		batchIDExpr, nextArg,
	)
	return fragment, []any{s.UserID}
}
