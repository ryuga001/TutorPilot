CREATE TABLE customers (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_name   VARCHAR(100) NOT NULL UNIQUE,
    first_name VARCHAR(50)  NOT NULL,
    last_name  VARCHAR(50)  NOT NULL,
    jwt_secret TEXT         NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(50) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    UNIQUE (customer_id, name)
);

CREATE TABLE privileges (
    id   INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL
);

CREATE TABLE role_privileges (
    role_id      INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    privilege_id INT NOT NULL REFERENCES privileges(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, privilege_id)
);

CREATE TABLE dashboard_users (
    id            INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id   INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    role_id       INT REFERENCES roles(id) ON DELETE SET NULL,
    email         VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    password_salt VARCHAR(255) NOT NULL DEFAULT gen_random_uuid()::text,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE email_templates (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    variables   JSONB        NOT NULL DEFAULT '{}'::jsonb,
    subject     VARCHAR(255) NOT NULL,
    body        TEXT         NOT NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name)
);

INSERT INTO customers (org_name, first_name, last_name)
VALUES ('TutorPilot', 'Super', 'Admin');

INSERT INTO roles (name, type, customer_id) VALUES
('Super Admin', 'super_admin', 1),
('Admin',       'admin',       1),
('User',        'user',        1);

-- Global privilege catalog.
INSERT INTO privileges (name, type) VALUES
('manage_dashboard_users', 'user'),
('manage_roles',           'role'),
('manage_privileges',      'privilege'),
('manage_email_templates', 'email_template'),
('view_dashboard',         'dashboard');

-- Grants for the TutorPilot tenant's roles.
-- Super Admin: every privilege.
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN privileges p
WHERE r.customer_id = 1 AND r.type = 'super_admin';

-- Admin: everything except privilege management.
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN privileges p
WHERE r.customer_id = 1 AND r.type = 'admin' AND p.name <> 'manage_privileges';

-- User: dashboard access only.
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN privileges p
WHERE r.customer_id = 1 AND r.type = 'user' AND p.name = 'view_dashboard';

INSERT INTO dashboard_users (customer_id, role_id, email, password_hash)
VALUES (
    1,
    (SELECT id FROM roles WHERE type = 'super_admin' AND customer_id = 1),
    'superadmin@tutorpilot.ai',
    'supersecurepasswordhash'
);

INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'welcome',
    'Welcome to TutorPilot',
    '{"name": "Recipient full name", "org_name": "Organization name", "login_url": "Dashboard login URL"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">Welcome, {{name}}!</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Your {{org_name}} workspace on TutorPilot is ready. Sign in to start managing your tutoring platform.
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0;">
                <tr>
                  <td style="border-radius:8px;background-color:#4f46e5;">
                    <a href="{{login_url}}" style="display:inline-block;padding:12px 28px;color:#ffffff;font-size:15px;font-weight:bold;text-decoration:none;">Go to Dashboard</a>
                  </td>
                </tr>
              </table>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                If the button does not work, copy this link into your browser:<br>{{login_url}}
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
    'password_reset',
    'Reset your TutorPilot password',
    '{"name": "Recipient full name", "otp": "One-time reset code", "expiry_minutes": "Minutes until the code expires"}'::jsonb,
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
              <h2 style="margin:0 0 16px 0;font-size:20px;">Password reset</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, we received a request to reset your TutorPilot password. Enter the code below to continue:
              </p>
              <div style="margin:24px 0;padding:16px;background-color:#f3f4f6;border-radius:8px;text-align:center;font-size:28px;font-weight:bold;letter-spacing:6px;color:#111827;">
                {{otp}}
              </div>
              <p style="margin:0 0 8px 0;font-size:14px;line-height:1.6;color:#4b5563;">
                This code expires in {{expiry_minutes}} minutes.
              </p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                If you did not request a password reset, you can safely ignore this email.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px;background-color:#f9fafb;text-align:center;color:#9ca3af;font-size:12px;">
              For your security, never share this code with anyone.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
);