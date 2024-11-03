-- name: GetEras :many
select *
from eras;

-- name: GetCurrEra :one
select *
from eras
where end_time = '2200/1/1';

-- TODO: this
-- name: InsertEra :one
--insert into eras (name, start_time, end_time, created_time, updated_time)
--values
--()
--returning *;

-- TODO: update query

-- It is intentional to have no query to delete an era.

