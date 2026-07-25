-- Removing the privileges cascades to role_privileges.
DELETE FROM privileges WHERE name IN ('course.create','course.view','course.edit','course.delete');
