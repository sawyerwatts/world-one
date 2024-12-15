-- name: GetEras :many
select *
from eras;

-- name: GetCurrEra :one
select *
from eras
where end_time = '2200/1/1';

-- name: InsertEra :one
insert into eras (name, start_time, end_time)
values           ($1,   $2,         $3)
returning *;

-- name: UpdateEra :one
update eras
set
		name = $2,
		start_time = $3,
		end_time = $4
where id = $1
		and update_time = $5
returning *;

-- name: DeleteEra :execrows
delete from eras
where id = $1;

-- name: TruncateEra :execrows
delete from eras;

