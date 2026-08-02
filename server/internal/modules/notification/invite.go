package notification

type MemberInvite struct {
	Name         string
	Role         string
	Email        string
	TempPassword string
	SignInURL    string
}

func InviteVars(invite MemberInvite) map[string]string {
	return map[string]string{
		"name":          invite.Name,
		"role":          invite.Role,
		"email":         invite.Email,
		"temp_password": invite.TempPassword,
		"sign_in_url":   invite.SignInURL,
	}
}
