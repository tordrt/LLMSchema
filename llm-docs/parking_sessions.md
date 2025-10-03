## parking_sessions

| Column | Type |
|--------|------|
| id | PK bigint NOT NULL |
| tenant_id | integer NOT NULL |
| parking_id | bigint NOT NULL |
| section_id | integer NOT NULL |
| lpn | text NOT NULL |
| country_code | varchar(3) NOT NULL |
| started_at | timestamptz NOT NULL |
| stopped_at | timestamptz |
| status | session_status (active, completed, cancelled) NOT NULL DEFAULT 'active'::session_status |
| created_at | timestamptz NOT NULL DEFAULT now() |
| updated_at | timestamptz NOT NULL DEFAULT now() |

### Index

- idx_unique_active_session_per_lpn on (lpn, country_code), unique

### References

- parking_id → parkings.id (many parking_sessions to one parkings)
- section_id → sections.id (many parking_sessions to one sections)
- tenant_id → tenants.id (many parking_sessions to one tenants)

