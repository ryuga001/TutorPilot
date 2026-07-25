INSERT INTO privileges (name, type) VALUES
('tutor.create', 'tutor'),
('tutor.view',   'tutor'),
('tutor.edit',   'tutor'),
('tutor.delete', 'tutor'),
('student.create', 'student'),
('student.view',   'student'),
('student.edit',   'student'),
('student.delete', 'student');

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY[
    'tutor.create','tutor.view','tutor.edit','tutor.delete',
    'student.create','student.view','student.edit','student.delete'
])
WHERE r.name = 'Admin'
ON CONFLICT DO NOTHING;
