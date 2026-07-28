ALTER TABLE students
    DROP CONSTRAINT IF EXISTS students_dashboard_user_id_fkey,
    DROP CONSTRAINT IF EXISTS students_pkey;

ALTER TABLE students
    ADD COLUMN id          INT GENERATED ALWAYS AS IDENTITY,
    ADD COLUMN customer_id INT,
    ADD COLUMN first_name  VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN last_name   VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN email       VARCHAR(150) NOT NULL DEFAULT '';

UPDATE students s
SET customer_id = du.customer_id,
    first_name = du.first_name, last_name = du.last_name, email = du.email
FROM dashboard_users du
WHERE du.id = s.dashboard_user_id;

ALTER TABLE students
    ALTER COLUMN customer_id SET NOT NULL,
    ADD CONSTRAINT students_customer_id_fkey FOREIGN KEY (customer_id)
        REFERENCES customers(id) ON DELETE CASCADE;
ALTER TABLE students ADD CONSTRAINT students_pkey PRIMARY KEY (id);
ALTER TABLE students ADD CONSTRAINT students_customer_id_email_key UNIQUE (customer_id, email);

ALTER TABLE batch_students ADD COLUMN new_student_id INT;
UPDATE batch_students bs SET new_student_id = s.id
FROM students s WHERE s.dashboard_user_id = bs.student_id;

ALTER TABLE batch_students DROP CONSTRAINT IF EXISTS batch_students_student_id_fkey;
ALTER TABLE batch_students DROP CONSTRAINT IF EXISTS batch_students_batch_id_student_id_key;
ALTER TABLE batch_students DROP COLUMN student_id;
ALTER TABLE batch_students RENAME COLUMN new_student_id TO student_id;
ALTER TABLE batch_students ALTER COLUMN student_id SET NOT NULL;
ALTER TABLE batch_students ADD CONSTRAINT batch_students_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
ALTER TABLE batch_students ADD CONSTRAINT batch_students_batch_id_student_id_key
    UNIQUE (batch_id, student_id);

DELETE FROM dashboard_users du
USING students s
WHERE s.dashboard_user_id = du.id
  AND du.password_hash = '$migration$disabled-until-password-reset';

ALTER TABLE students DROP COLUMN dashboard_user_id;

ALTER TABLE tutors
    DROP CONSTRAINT IF EXISTS tutors_dashboard_user_id_fkey,
    DROP CONSTRAINT IF EXISTS tutors_pkey;

ALTER TABLE tutors
    ADD COLUMN id          INT GENERATED ALWAYS AS IDENTITY,
    ADD COLUMN customer_id INT,
    ADD COLUMN first_name  VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN last_name   VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN email       VARCHAR(150) NOT NULL DEFAULT '';

UPDATE tutors t
SET customer_id = du.customer_id,
    first_name = du.first_name, last_name = du.last_name, email = du.email
FROM dashboard_users du
WHERE du.id = t.dashboard_user_id;

ALTER TABLE tutors
    ALTER COLUMN customer_id SET NOT NULL,
    ADD CONSTRAINT tutors_customer_id_fkey FOREIGN KEY (customer_id)
        REFERENCES customers(id) ON DELETE CASCADE;
ALTER TABLE tutors ADD CONSTRAINT tutors_pkey PRIMARY KEY (id);
ALTER TABLE tutors ADD CONSTRAINT tutors_customer_id_email_key UNIQUE (customer_id, email);

ALTER TABLE batch_module_tutors ADD COLUMN new_tutor_id INT;
UPDATE batch_module_tutors bmt SET new_tutor_id = t.id
FROM tutors t WHERE t.dashboard_user_id = bmt.tutor_id;

ALTER TABLE batch_module_tutors DROP COLUMN tutor_id;
ALTER TABLE batch_module_tutors RENAME COLUMN new_tutor_id TO tutor_id;
ALTER TABLE batch_module_tutors ALTER COLUMN tutor_id SET NOT NULL;
ALTER TABLE batch_module_tutors ADD CONSTRAINT batch_module_tutors_tutor_id_fkey
    FOREIGN KEY (tutor_id) REFERENCES tutors(id) ON DELETE CASCADE;

ALTER TABLE batch_tutors ADD COLUMN new_tutor_id INT;
UPDATE batch_tutors bt SET new_tutor_id = t.id
FROM tutors t WHERE t.dashboard_user_id = bt.tutor_id;

ALTER TABLE batch_tutors DROP CONSTRAINT IF EXISTS batch_tutors_tutor_id_fkey;
ALTER TABLE batch_tutors DROP CONSTRAINT IF EXISTS batch_tutors_batch_id_tutor_id_key;
ALTER TABLE batch_tutors DROP COLUMN tutor_id;
ALTER TABLE batch_tutors RENAME COLUMN new_tutor_id TO tutor_id;
ALTER TABLE batch_tutors ALTER COLUMN tutor_id SET NOT NULL;
ALTER TABLE batch_tutors ADD CONSTRAINT batch_tutors_tutor_id_fkey
    FOREIGN KEY (tutor_id) REFERENCES tutors(id) ON DELETE CASCADE;
ALTER TABLE batch_tutors ADD CONSTRAINT batch_tutors_batch_id_tutor_id_key
    UNIQUE (batch_id, tutor_id);

ALTER TABLE lectures ADD COLUMN new_tutor_id INT;
UPDATE lectures l SET new_tutor_id = t.id
FROM tutors t WHERE t.dashboard_user_id = l.tutor_id;

ALTER TABLE lectures DROP COLUMN tutor_id;
ALTER TABLE lectures RENAME COLUMN new_tutor_id TO tutor_id;
ALTER TABLE lectures ADD CONSTRAINT lectures_tutor_id_fkey
    FOREIGN KEY (tutor_id) REFERENCES tutors(id) ON DELETE SET NULL;

DELETE FROM dashboard_users du
USING tutors t
WHERE t.dashboard_user_id = du.id
  AND du.password_hash = '$migration$disabled-until-password-reset';


DELETE FROM role_privileges
WHERE role_id IN (SELECT id FROM roles WHERE name IN ('Tutor', 'Student'));
DELETE FROM roles WHERE name IN ('Tutor', 'Student');

DELETE FROM role_privileges
WHERE privilege_id IN (
    SELECT id FROM privileges WHERE name IN (
        'portal.access', 'lecture.join', 'lecture.publish', 'recording.view',
        'drive.upload', 'self.profile.view', 'self.profile.edit', 'manage_members'
    )
);
DELETE FROM privileges WHERE name IN (
    'portal.access', 'lecture.join', 'lecture.publish', 'recording.view',
    'drive.upload', 'self.profile.view', 'self.profile.edit', 'manage_members'
);

ALTER TABLE dashboard_users DROP COLUMN first_name, DROP COLUMN last_name;
