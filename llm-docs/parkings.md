## parkings

| Column | Type |
|--------|------|
| id | PK bigint NOT NULL |
| tenant_id | integer NOT NULL |
| facility_id | integer NOT NULL |
| lpn | text NOT NULL |
| status | parking_status (active, completed, cancelled) NOT NULL DEFAULT 'active'::parking_status |
| country_code | varchar(3) NOT NULL |
| entry_at | timestamptz NOT NULL |
| exit_at | timestamptz |
| created_at | timestamptz NOT NULL DEFAULT now() |
| updated_at | timestamptz NOT NULL DEFAULT now() |

### Index

- idx_unique_active_parking_per_lpn on (lpn, country_code), unique

### References

- facility_id → facilities.id (many parkings to one facilities)
- tenant_id → tenants.id (many parkings to one tenants)

### Referenced by

- parking_sessions.parking_id → id (many parking_sessions to one parkings)

