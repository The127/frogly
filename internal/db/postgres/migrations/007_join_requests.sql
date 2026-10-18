-- +migrate Up

alter table projects drop constraint projects_joining_check;
alter table projects
    add constraint projects_joining_check check (joining in ('open', 'request', 'closed'));

-- +migrate Down

-- the version trigger only lets an update through that writes the next
-- version
update projects set joining = 'closed', version = version + 1 where joining = 'request';

alter table projects drop constraint projects_joining_check;
alter table projects
    add constraint projects_joining_check check (joining in ('open', 'closed'));
