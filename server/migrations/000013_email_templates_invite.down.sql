-- Restores the member_invite template exactly as seeded in 000011, including the
-- {{activation_url}} / {{expires_in}} variables that no producer can fill. This
-- is a faithful rollback, not a correct template -- reverting past 000013 puts
-- the dead activation-link body back.

UPDATE email_templates
SET subject = 'You have been invited to {{org_name}} on TutorPilot',
    variables = '{"name": "Member name", "org_name": "Organization name", "email": "Login email", "temp_password": "Temporary password", "portal_url": "Organization portal URL", "activation_url": "Set-password link", "expires_in": "Invitation lifetime"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">Welcome to {{org_name}}</h2>
              <p style="margin:0 0 24px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, an account has been created for you. Choose a password to get started.
              </p>
              <p style="margin:0 0 28px 0;">
                <a href="{{activation_url}}"
                   style="display:inline-block;background-color:#4f46e5;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-size:15px;">
                  Set your password
                </a>
              </p>
              <p style="margin:0 0 16px 0;font-size:13px;line-height:1.6;color:#6b7280;">
                This link expires in {{expires_in}}.
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%;margin:20px 0;border-collapse:collapse;background-color:#f9fafb;border-radius:8px;">
                <tr>
                  <td style="padding:14px 16px 4px 16px;font-size:13px;color:#6b7280;">
                    If the link has expired, sign in at
                    <a href="{{portal_url}}" style="color:#4f46e5;">{{portal_url}}</a> with these
                    details and you will be asked to set a password:
                  </td>
                </tr>
                <tr>
                  <td style="padding:8px 16px;font-size:14px;color:#111827;">
                    Email: <strong>{{email}}</strong><br />
                    Temporary password: <strong>{{temp_password}}</strong>
                  </td>
                </tr>
              </table>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                Sign in only at your organization''s address above. Do not share this email.
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
