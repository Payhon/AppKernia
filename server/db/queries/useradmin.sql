-- name: TenantManagedUserExists :one
SELECT EXISTS (
    SELECT 1
    FROM iam.tenant_members AS tm
    JOIN iam.users AS u ON u.id = tm.user_id
    WHERE tm.tenant_id = $1
      AND tm.user_id = $2
      AND u.deleted_at IS NULL
);
