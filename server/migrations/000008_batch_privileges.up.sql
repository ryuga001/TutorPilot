INSERT INTO privileges (name, type) VALUES
('batch.create', 'batch'),
('batch.view',   'batch'),
('batch.edit',   'batch'),
('batch.delete', 'batch');

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (ARRAY['batch.create','batch.view','batch.edit','batch.delete'])
WHERE r.name = 'Admin'
ON CONFLICT DO NOTHING;
