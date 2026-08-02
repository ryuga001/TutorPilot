-- Rewrites the member_invite template seeded in 000011.
--
-- That version rendered a "Set your password" button pointing at {{activation_url}}
-- and a "this link expires in {{expires_in}}" line. The activation-token flow was
-- removed (there is no /auth/activate route), so neither variable can ever be
-- filled -- the template would render a dead button. This replaces it with the
-- flow that actually exists: sign in with a temporary password, change it after.
--
-- {{role}} is new, so one template serves tutor, student and CSV-import invites
-- instead of three blocks of hardcoded HTML in Go string literals.

UPDATE email_templates
SET subject = 'Your TutorPilot {{role}} account',
    variables = '{"name": "Member name", "role": "tutor or student", "email": "Login email", "temp_password": "Temporary password", "sign_in_url": "Sign-in page URL"}'::jsonb,
    body = '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="background-color:#4f46e5;padding:28px 40px;text-align:center;">
              <h1 style="margin:0;color:#ffffff;font-size:22px;">TutorPilot</h1>
            </td>
          </tr>
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Your {{role}} account is ready</h2>
              <p style="margin:0 0 24px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, an account has been created for you on TutorPilot. Sign in with the
                temporary password below, then change it from your profile.
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%;margin:0 0 28px 0;border-collapse:collapse;background-color:#f9fafb;border-radius:8px;">
                <tr>
                  <td style="padding:16px;font-size:14px;color:#111827;">
                    Email: <strong>{{email}}</strong><br />
                    Temporary password: <strong>{{temp_password}}</strong>
                  </td>
                </tr>
              </table>
              <p style="margin:0 0 28px 0;">
                <a href="{{sign_in_url}}"
                   style="display:inline-block;background-color:#4f46e5;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-size:15px;">
                  Sign in
                </a>
              </p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                This password is only for your first sign-in. Do not share this email.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              You are receiving this email because an account was created for you on TutorPilot.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
WHERE customer_id = 1 AND name = 'member_invite';
