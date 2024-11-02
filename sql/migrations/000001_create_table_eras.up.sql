-- Eras represent stuff.
create table if not exists eras(
		id bigserial primary key,
		name text not null unique,
		start_time timestamptz not null,
		end_time timestamptz not null,
		created_time timestamptz not null,
		updated_time timestamptz not null
);

