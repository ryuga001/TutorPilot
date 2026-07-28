ALTER TABLE lectures
    ADD COLUMN recording_status           VARCHAR(16) NOT NULL DEFAULT 'none',
    ADD COLUMN recording_object_key       TEXT,
    ADD COLUMN recording_node_id          INT REFERENCES batch_drive_nodes(id) ON DELETE SET NULL,
    ADD COLUMN recording_duration_seconds INT,
    ADD COLUMN recording_size_bytes       BIGINT,
    ADD COLUMN actual_start_at            TIMESTAMPTZ,
    ADD CONSTRAINT lectures_status_chk
        CHECK (status IN ('scheduled', 'live', 'ended', 'cancelled')),
    ADD CONSTRAINT lectures_recording_status_chk
        CHECK (recording_status IN ('none', 'starting', 'recording', 'processing', 'ready', 'failed'));

COMMENT ON COLUMN lectures.start_time IS 'Scheduled start';
COMMENT ON COLUMN lectures.actual_start_at IS 'When the lecture was actually started';

CREATE TABLE lecture_attendance (
                                    id              INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                    lecture_id      BIGINT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
                                    user_id         INT NOT NULL REFERENCES dashboard_users(id) ON DELETE CASCADE,
                                    subject_type    VARCHAR(16) NOT NULL,
                                    subject_id      INT,
                                    display_name    VARCHAR(200) NOT NULL DEFAULT '',
                                    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
                                    left_at         TIMESTAMPTZ,
                                    seconds_present INT,
                                    CONSTRAINT lecture_attendance_session_key UNIQUE (lecture_id, user_id, joined_at)
);

ALTER TABLE batch_drive_nodes
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;



INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'member_invite',
    'You have been invited to {{org_name}} on TutorPilot',
    '{"name": "Member name", "org_name": "Organization name", "email": "Login email", "temp_password": "Temporary password", "portal_url": "Organization portal URL", "activation_url": "Set-password link", "expires_in": "Invitation lifetime"}'::jsonb,
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
),
(
    1,
    'member_credentials_reset',
    'Your {{org_name}} sign-in details have been reissued',
    '{"name": "Member name", "org_name": "Organization name", "email": "Login email", "temp_password": "Temporary password", "portal_url": "Organization portal URL", "activation_url": "Set-password link", "expires_in": "Invitation lifetime"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">Your sign-in details changed</h2>
              <p style="margin:0 0 24px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, an administrator at {{org_name}} reissued your credentials. Your
                previous password and any earlier invitation no longer work.
              </p>
              <p style="margin:0 0 28px 0;">
                <a href="{{activation_url}}"
                   style="display:inline-block;background-color:#4f46e5;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-size:15px;">
                  Set a new password
                </a>
              </p>
              <p style="margin:0 0 16px 0;font-size:13px;line-height:1.6;color:#6b7280;">
                This link expires in {{expires_in}}.
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%;margin:20px 0;border-collapse:collapse;background-color:#f9fafb;border-radius:8px;">
                <tr>
                  <td style="padding:14px 16px 4px 16px;font-size:13px;color:#6b7280;">
                    Or sign in at <a href="{{portal_url}}" style="color:#4f46e5;">{{portal_url}}</a>
                    with these details and set a password when prompted:
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
                If you did not expect this, contact your organization administrator.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              You are receiving this email because your TutorPilot credentials were reissued.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
);
