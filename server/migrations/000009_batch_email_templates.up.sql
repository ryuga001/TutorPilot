INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'batch_tutor_assignment',
    'You have been assigned to teach a module on TutorPilot',
    '{"name": "Tutor name", "batch_name": "Batch name", "course_name": "Course name", "module_title": "Module title", "start_date": "Assignment start date", "expected_end_date": "Assignment expected end date"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">New teaching assignment</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, you have been assigned to teach the module <strong>{{module_title}}</strong>
                as part of <strong>{{batch_name}}</strong> ({{course_name}}).
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%;margin:20px 0;border-collapse:collapse;">
                <tr>
                  <td style="padding:8px 0;font-size:14px;color:#6b7280;">Start date</td>
                  <td style="padding:8px 0;font-size:14px;color:#111827;text-align:right;">{{start_date}}</td>
                </tr>
                <tr>
                  <td style="padding:8px 0;font-size:14px;color:#6b7280;border-top:1px solid #f3f4f6;">Expected end date</td>
                  <td style="padding:8px 0;font-size:14px;color:#111827;text-align:right;border-top:1px solid #f3f4f6;">{{expected_end_date}}</td>
                </tr>
              </table>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                Reach out to your organization admin if you have any questions about this assignment.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              You are receiving this email because you were assigned to a batch on TutorPilot.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
),
(
    1,
    'batch_student_enrollment',
    'You have been enrolled in a batch on TutorPilot',
    '{"name": "Student name", "batch_name": "Batch name", "course_name": "Course name"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">You&#39;re enrolled!</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, you have been enrolled in <strong>{{batch_name}}</strong> ({{course_name}}).
              </p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                Your organization admin will share further details about schedule and access.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              You are receiving this email because you were enrolled in a batch on TutorPilot.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
);
