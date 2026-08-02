package notification

import "testing"

func TestInviteVarsCoversEveryTemplatePlaceholder(t *testing.T) {
	vars := InviteVars(MemberInvite{
		Name: "Ada", Role: "tutor", Email: "ada@example.com",
		TempPassword: "hunter2", SignInURL: "https://example.test/login",
	})

	for _, key := range []string{"name", "role", "email", "temp_password", "sign_in_url"} {
		if _, ok := vars[key]; !ok {
			t.Errorf("InviteVars is missing %q, which the member_invite template references", key)
		}
	}
}

func TestInviteVarsCarriesValuesThrough(t *testing.T) {
	in := MemberInvite{
		Name: "Ada", Role: "student", Email: "ada@example.com",
		TempPassword: "hunter2", SignInURL: "https://example.test/login",
	}

	vars := InviteVars(in)

	if vars["temp_password"] != in.TempPassword {
		t.Errorf("temp_password = %q, want %q: it exists nowhere else in the system", vars["temp_password"], in.TempPassword)
	}
	if vars["role"] != "student" {
		t.Errorf("role = %q, want %q", vars["role"], "student")
	}
}
