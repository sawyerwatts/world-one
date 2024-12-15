begin;

drop trigger if exists trigger_update_eras on eras.update_time;

drop table if exists eras;

drop extension if exists moddatetime;

commit;
