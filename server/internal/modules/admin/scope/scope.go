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
