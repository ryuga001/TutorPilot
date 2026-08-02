package scope

import (
	"strings"
	"testing"
)

func TestBatchPredicateBindsExactlyOneArgument(t *testing.T) {
	s := Scope{CustomerID: 1, UserID: 42}

	frag, args := s.BatchPredicate("l.batch_id", 3)

	if len(args) != 1 {
		t.Fatalf("got %d args, want 1: the fragment reuses $n for every reference", len(args))
	}
	if args[0] != 42 {
		t.Errorf("args[0] = %v, want the caller's user id 42", args[0])
	}
	if strings.Contains(frag, "$4") {
		t.Errorf("fragment referenced a second placeholder; it must reuse $3 only:\n%s", frag)
	}
	if !strings.Contains(frag, "$3") {
		t.Errorf("fragment did not use the requested placeholder index:\n%s", frag)
	}
}

func TestBatchPredicateHonoursPlaceholderOffset(t *testing.T) {
	s := Scope{UserID: 7}

	frag, _ := s.BatchPredicate("l.batch_id", 9)

	if !strings.Contains(frag, "$9") {
		t.Errorf("fragment ignored nextArg=9:\n%s", frag)
	}
	if strings.Contains(frag, "$1 ") || strings.Contains(frag, "$2 ") {
		t.Errorf("fragment used a hardcoded low placeholder:\n%s", frag)
	}
}

func TestBatchPredicateSubstitutesTheBatchColumn(t *testing.T) {
	s := Scope{UserID: 1}

	frag, _ := s.BatchPredicate("b.id", 2)

	if !strings.Contains(frag, "bt.batch_id = b.id") {
		t.Errorf("tutor branch did not use the caller's column expression:\n%s", frag)
	}
	if !strings.Contains(frag, "bs.batch_id = b.id") {
		t.Errorf("student branch did not use the caller's column expression:\n%s", frag)
	}
}

func TestBatchPredicateCoversTutorStudentAndAdmin(t *testing.T) {
	frag, _ := Scope{UserID: 1}.BatchPredicate("l.batch_id", 2)

	if !strings.Contains(frag, "batch_tutors") {
		t.Error("no tutor-membership branch: an assigned tutor would be denied their own batch")
	}
	if !strings.Contains(frag, "batch_students") {
		t.Error("no student-membership branch: an enrolled student would be denied their own batch")
	}
	if !strings.Contains(frag, "NOT EXISTS") {
		t.Error("no admin fallthrough: an admin who is neither tutor nor student would see nothing")
	}
}

func TestBatchPredicateStartsAsAConjunction(t *testing.T) {
	frag, _ := Scope{UserID: 1}.BatchPredicate("l.batch_id", 2)

	if !strings.HasPrefix(strings.TrimSpace(frag), "AND") {
		t.Errorf("fragment must append to an existing WHERE clause, got:\n%s", frag)
	}
}
