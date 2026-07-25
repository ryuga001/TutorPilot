-- System email-verification OTP template (system tenant, customer_id = 1).
-- Placeholders: {{name}}, {{otp}}, {{expiry_minutes}}, {{verify_link}}.
INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'email_verification',
    'Verify your TutorPilot email',
    '{"name": "Recipient name", "otp": "One-time verification code", "expiry_minutes": "Minutes until the code expires", "verify_link": "Verification page link"}'::jsonb,
    '<!DOCTYPE html>
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">Verify your email</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, use the code below to verify your email and continue setting up your TutorPilot account:
              </p>
              <div style="margin:24px 0;padding:16px;background-color:#f3f4f6;border-radius:8px;text-align:center;font-size:28px;font-weight:bold;letter-spacing:6px;color:#111827;">
                {{otp}}
              </div>
              <p style="margin:0 0 8px 0;font-size:14px;line-height:1.6;color:#4b5563;">
                This code expires in {{expiry_minutes}} minutes.
              </p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                You can also open the verification page directly:<br><a href="{{verify_link}}" style="color:#4f46e5;">{{verify_link}}</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              If you did not request this, you can safely ignore this email.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
);
