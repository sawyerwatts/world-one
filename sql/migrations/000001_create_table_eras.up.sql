begin;

create extension if not exists moddatetime;

create table if not exists eras(
		id bigint generated always as identity primary key,
		name text not null unique,
		start_time timestamptz not null,
		end_time timestamptz not null,
		create_time timestamptz not null default now(),
		update_time timestamptz not null default now()
);

create trigger trig_era_modatetime_to_update_time
	before update on eras
	for each row
	execute procedure moddatetime(update_time);

commit;
