INSERT INTO privileges (name, type) VALUES
('course.create', 'course'),
('course.view',   'course'),
('course.edit',   'course'),
('course.delete', 'course');

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY['course.create','course.view','course.edit','course.delete'])
WHERE r.name = 'Admin'
ON CONFLICT DO NOTHING;