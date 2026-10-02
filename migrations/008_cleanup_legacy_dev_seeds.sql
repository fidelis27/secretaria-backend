DELETE FROM users
WHERE email = 'demo-active@example.com'
  AND NOT EXISTS (
      SELECT 1
      FROM member_groups
      WHERE member_groups.user_id = users.id
  );

DELETE FROM institutions
WHERE id = 'transfer-origin'
  AND NOT EXISTS (
      SELECT 1
      FROM `groups`
      WHERE `groups`.institution_id = institutions.id
  )
  AND NOT EXISTS (
      SELECT 1
      FROM students
      WHERE students.institution_id = institutions.id
  )
  AND NOT EXISTS (
      SELECT 1
      FROM enrollments
      WHERE enrollments.institution_id = institutions.id
  )
  AND NOT EXISTS (
      SELECT 1
      FROM audit_event_institutions
      WHERE audit_event_institutions.institution_id = institutions.id
  );
