-- Tutors and students stop duplicating identity. A tutor/student *is* a
-- dashboard_users row (email, password, name) plus a small extras row holding
-- only what dashboard_users doesn't have (phone, designation, address, photo).
-- The extras row's primary key IS the dashboard_users id — there is no separate
-- surrogate id, so a tutor's id, their JWT uid, and every FK that points at them
-- (batch_tutors.tutor_id, lectures.tutor_id, ...) are all the same integer.
--
-- This also lets tutors/students log in through the exact same path as an admin:
-- dashboard_users already carries role_id/role_privileges, so no separate
-- subject-type or per-row scoping machinery is needed for authentication itself.

-- === Part 1: privileges + roles, for every existing tenant =================
-- New tenants already get these from newTenantRoles in
-- internal/modules/auth/repository.go; existing tenants need a backfill.

INSERT INTO privileges (name, type) VALUES
('portal.access',     'portal'),
('lecture.join',      'lecture'),
('lecture.publish',   'lecture'),
('recording.view',    'lecture'),
('drive.upload',      'batch'),
('self.profile.view', 'self'),
('self.profile.edit', 'self'),
('manage_members',    'admin')
ON CONFLICT (name) DO NOTHING;

-- Existing tenants' Admin roles were granted a fixed snapshot of the catalog at
-- registration time, so privileges added since then must be granted explicitly.
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY[
    'portal.access', 'lecture.join', 'lecture.publish', 'recording.view',
    'drive.upload', 'self.profile.view', 'self.profile.edit', 'manage_members'
])
WHERE r.name = 'Admin'
ON CONFLICT DO NOTHING;

INSERT INTO roles (name, type, customer_id)
SELECT 'Tutor', 'tutor', c.id FROM customers c
ON CONFLICT (customer_id, name) DO NOTHING;

INSERT INTO roles (name, type, customer_id)
SELECT 'Student', 'student', c.id FROM customers c
ON CONFLICT (customer_id, name) DO NOTHING;

-- A student looks at their own courses/batches, joins their lectures, watches
-- recordings, edits their own profile. Mirrors studentPrivileges in
-- internal/modules/auth/repository.go.
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY[
    'portal.access', 'self.profile.view', 'self.profile.edit',
    'course.view', 'batch.view', 'lecture.view', 'lecture.join', 'recording.view'
])
WHERE r.name = 'Student'
ON CONFLICT DO NOTHING;

-- A tutor gets everything a student gets, plus running lectures and managing
-- batch material. Mirrors tutorPrivileges (student ∪ tutorOnlyPrivileges).
INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY[
    'portal.access', 'self.profile.view', 'self.profile.edit',
    'course.view', 'batch.view', 'lecture.view', 'lecture.join', 'recording.view',
    'view_dashboard', 'student.view', 'drive.upload',
    'lecture.create', 'lecture.edit', 'lecture.control', 'lecture.publish'
])
WHERE r.name = 'Tutor'
ON CONFLICT DO NOTHING;

-- === Part 2: dashboard_users gains a name =====================================
-- Every principal type now needs its own name; previously only the tenant
-- contact (customers.first_name/last_name) had one.

ALTER TABLE dashboard_users
    ADD COLUMN first_name VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN last_name  VARCHAR(100) NOT NULL DEFAULT '';

-- Best-effort backfill for existing admins: the org's contact name is the
-- closest thing to a personal name recorded for them today.
UPDATE dashboard_users du
SET first_name = c.first_name, last_name = c.last_name
FROM customers c
WHERE c.id = du.customer_id AND du.first_name = '';

-- === Part 3: tutors becomes a dashboard_users extension =======================
-- Note: dashboard_users.email is unique across the whole installation (a
-- pre-existing constraint, not new here). If any two tutors in different
-- tenants happen to share an email address, this step fails loudly rather than
-- silently merging or dropping a row — which is the correct outcome; resolve
-- the collision by hand and re-run.

ALTER TABLE tutors ADD COLUMN dashboard_user_id INT;

WITH inserted AS (
    INSERT INTO dashboard_users
        (customer_id, role_id, email, password_hash, password_salt, first_name, last_name)
    SELECT t.customer_id, r.id, t.email,
           '$migration$disabled-until-password-reset',
           gen_random_uuid()::text,
           t.first_name, t.last_name
    FROM tutors t
    JOIN roles r ON r.customer_id = t.customer_id AND r.name = 'Tutor'
    RETURNING id, customer_id, email
)
UPDATE tutors t
SET dashboard_user_id = inserted.id
FROM inserted
WHERE t.customer_id = inserted.customer_id AND t.email = inserted.email;

ALTER TABLE tutors ALTER COLUMN dashboard_user_id SET NOT NULL;

-- Remap every foreign key that pointed at tutors.id onto the new
-- dashboard_user_id, before the old id disappears.
ALTER TABLE batch_module_tutors ADD COLUMN tutor_dashboard_user_id INT;
UPDATE batch_module_tutors bmt SET tutor_dashboard_user_id = t.dashboard_user_id
FROM tutors t WHERE t.id = bmt.tutor_id;

ALTER TABLE batch_tutors ADD COLUMN tutor_dashboard_user_id INT;
UPDATE batch_tutors bt SET tutor_dashboard_user_id = t.dashboard_user_id
FROM tutors t WHERE t.id = bt.tutor_id;

ALTER TABLE lectures ADD COLUMN tutor_dashboard_user_id INT;
UPDATE lectures l SET tutor_dashboard_user_id = t.dashboard_user_id
FROM tutors t WHERE t.id = l.tutor_id;

DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT c.conname, c.conrelid::regclass::text AS tbl
        FROM pg_constraint c
        JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
        WHERE c.contype = 'f' AND a.attname = 'tutor_id'
          AND c.conrelid IN ('batch_module_tutors'::regclass, 'batch_tutors'::regclass, 'lectures'::regclass)
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);
    END LOOP;
END $$;

ALTER TABLE batch_module_tutors DROP COLUMN tutor_id;
ALTER TABLE batch_module_tutors RENAME COLUMN tutor_dashboard_user_id TO tutor_id;
ALTER TABLE batch_module_tutors ALTER COLUMN tutor_id SET NOT NULL;

ALTER TABLE batch_tutors DROP COLUMN tutor_id CASCADE; -- cascades the old UNIQUE(batch_id, tutor_id)
ALTER TABLE batch_tutors RENAME COLUMN tutor_dashboard_user_id TO tutor_id;
ALTER TABLE batch_tutors ALTER COLUMN tutor_id SET NOT NULL;
ALTER TABLE batch_tutors ADD CONSTRAINT batch_tutors_batch_id_tutor_id_key UNIQUE (batch_id, tutor_id);

ALTER TABLE lectures DROP COLUMN tutor_id;
ALTER TABLE lectures RENAME COLUMN tutor_dashboard_user_id TO tutor_id;

-- Drop the columns that moved to dashboard_users, then move the primary key.
-- customer_id goes too: it's available via the dashboard_user_id join, and
-- dashboard_users -> customers already cascades tutors' deletion transitively
-- (customers delete -> dashboard_users delete -> tutors delete), so tenant
-- integrity does not depend on tutors carrying its own copy.
ALTER TABLE tutors DROP CONSTRAINT tutors_pkey;
ALTER TABLE tutors DROP COLUMN id;
ALTER TABLE tutors DROP COLUMN first_name;
ALTER TABLE tutors DROP COLUMN last_name;
ALTER TABLE tutors DROP COLUMN email; -- also drops UNIQUE(customer_id, email)
ALTER TABLE tutors DROP COLUMN customer_id;

ALTER TABLE tutors
    ADD CONSTRAINT tutors_pkey PRIMARY KEY (dashboard_user_id),
    ADD CONSTRAINT tutors_dashboard_user_id_fkey FOREIGN KEY (dashboard_user_id)
        REFERENCES dashboard_users(id) ON DELETE CASCADE;

ALTER TABLE batch_module_tutors
    ADD CONSTRAINT batch_module_tutors_tutor_id_fkey FOREIGN KEY (tutor_id)
        REFERENCES tutors(dashboard_user_id) ON DELETE CASCADE;

ALTER TABLE batch_tutors
    ADD CONSTRAINT batch_tutors_tutor_id_fkey FOREIGN KEY (tutor_id)
        REFERENCES tutors(dashboard_user_id) ON DELETE CASCADE;

ALTER TABLE lectures
    ADD CONSTRAINT lectures_tutor_id_fkey FOREIGN KEY (tutor_id)
        REFERENCES tutors(dashboard_user_id) ON DELETE SET NULL;

-- === Part 4: students becomes a dashboard_users extension, same pattern ======

ALTER TABLE students ADD COLUMN dashboard_user_id INT;

WITH inserted AS (
    INSERT INTO dashboard_users
        (customer_id, role_id, email, password_hash, password_salt, first_name, last_name)
    SELECT s.customer_id, r.id, s.email,
           '$migration$disabled-until-password-reset',
           gen_random_uuid()::text,
           s.first_name, s.last_name
    FROM students s
    JOIN roles r ON r.customer_id = s.customer_id AND r.name = 'Student'
    RETURNING id, customer_id, email
)
UPDATE students s
SET dashboard_user_id = inserted.id
FROM inserted
WHERE s.customer_id = inserted.customer_id AND s.email = inserted.email;

ALTER TABLE students ALTER COLUMN dashboard_user_id SET NOT NULL;

ALTER TABLE batch_students ADD COLUMN student_dashboard_user_id INT;
UPDATE batch_students bs SET student_dashboard_user_id = s.dashboard_user_id
FROM students s WHERE s.id = bs.student_id;

DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT c.conname, c.conrelid::regclass::text AS tbl
        FROM pg_constraint c
        JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
        WHERE c.contype = 'f' AND a.attname = 'student_id'
          AND c.conrelid = 'batch_students'::regclass
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);
    END LOOP;
END $$;

ALTER TABLE batch_students DROP COLUMN student_id CASCADE; -- cascades the old UNIQUE(batch_id, student_id)
ALTER TABLE batch_students RENAME COLUMN student_dashboard_user_id TO student_id;
ALTER TABLE batch_students ALTER COLUMN student_id SET NOT NULL;
ALTER TABLE batch_students ADD CONSTRAINT batch_students_batch_id_student_id_key UNIQUE (batch_id, student_id);

ALTER TABLE students DROP CONSTRAINT students_pkey;
ALTER TABLE students DROP COLUMN id;
ALTER TABLE students DROP COLUMN first_name;
ALTER TABLE students DROP COLUMN last_name;
ALTER TABLE students DROP COLUMN email;
ALTER TABLE students DROP COLUMN customer_id;

ALTER TABLE students
    ADD CONSTRAINT students_pkey PRIMARY KEY (dashboard_user_id),
    ADD CONSTRAINT students_dashboard_user_id_fkey FOREIGN KEY (dashboard_user_id)
        REFERENCES dashboard_users(id) ON DELETE CASCADE;

ALTER TABLE batch_students
    ADD CONSTRAINT batch_students_student_id_fkey FOREIGN KEY (student_id)
        REFERENCES students(dashboard_user_id) ON DELETE CASCADE;
