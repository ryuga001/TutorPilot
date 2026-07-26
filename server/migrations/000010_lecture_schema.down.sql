DELETE FROM role_privileges
WHERE privilege_id IN (
    SELECT id
    FROM privileges
    WHERE name IN (
           'lecture.create',
           'lecture.view',
           'lecture.edit',
           'lecture.delete',
           'lector.control'

    )
);

DELETE FROM privileges
WHERE name IN (
       'lecture.create',
       'lecture.view',
       'lecture.edit',
       'lecture.delete',
              'lector.control'
);

DROP TABLE IF EXISTS lectures;