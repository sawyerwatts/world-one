-- name: GetEras :many
select *
from eras;

-- name: GetCurrEra :one
select *
from eras
where end_time = '2200/1/1';

-- TODO: finish era db ops

